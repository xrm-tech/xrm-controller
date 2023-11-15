package ovirt

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/juju/fslock"
	cp "github.com/otiai10/copy"
	ovirtsdk4 "github.com/ovirt/go-ovirt"

	"github.com/xrm-tech/xrm-controller/pkg/ayaml"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	ErrStorageRemapEmptyResult = errors.New("dr_import_storages remap result epmty")
	ErrTemplateDirNotExist     = errors.New("ovirt template dir not exist")
	ErrDirAlreadyExist         = errors.New("dir already exist")
	ErrVarFileNotExist         = errors.New("var file not exist")
	// ansible
	ansibleDrTag            = "generate_mapping"
	ansibleGeneratePlaybook = "dr_generate.yml"
	ansibleFailoverPlaybook = "dr_failover.yml"
	ansibleFailbackPlaybook = "dr_failback.yml"
	ansibleDrVarsFile       = "disaster_recovery_vars.yml"
	ansibleDrPwdFile        = "ovirt_passwords.yml"

	StorageNFS   string = "nfs"
	StorageFCP   string = "fcp"
	StorageISCSI string = "iscsi"
)

func validateOvirtCon(url string, insecure bool, caFile, username, password string) error {
	builder := ovirtsdk4.NewConnectionBuilder().
		URL(url).
		Username(username).
		Password(password).
		Insecure(insecure).
		Timeout(time.Second * 10)

	if caFile != "" {
		builder = builder.CAFile(caFile)
	}

	if conn, err := builder.Build(); err == nil {
		defer conn.Close()

		if err = conn.Test(); err == nil {
			_, err = conn.SystemService().DataCentersService().List().Send()
		}

		return err
	} else {
		return err
	}
}

func StripStorageDomains(s []StorageMap) []StorageMap {
	pos := 0
	exist := make(map[string]bool)
	iSCSIexist := make(map[string]StorageMap)
	for i := 0; i < len(s); i++ {
		if s[i].PrimaryType == StorageNFS {
			s[i].PrimaryPath = strings.TrimRight(s[i].PrimaryPath, "/")
			s[i].SecondaryPath = strings.TrimRight(s[i].SecondaryPath, "/")
		}
		// cleanup from dulpicates
		key := s[i].PrimaryType + "://" + s[i].PrimaryId + ":" + s[i].PrimaryAddr + ":" + s[i].PrimaryPath + ":" + s[i].PrimaryPort
		if s[i].PrimaryType == StorageISCSI {
			if len(s[i].Targets) > 0 {
				if m, ok := iSCSIexist[key]; ok {
					for k, v := range s[i].Targets {
						m.Targets[k] = v
					}
					iSCSIexist[key] = m
				} else {
					iSCSIexist[key] = s[i]
				}
			}
		} else if _, ok := exist[key]; !ok {
			s[pos] = s[i]
			pos++
			exist[key] = true
		}
	}
	s = s[:pos]

	if len(iSCSIexist) > 0 {
		for _, m := range iSCSIexist {
			s = append(s, m)
		}
	}

	return s
}

type KV struct {
	K, V string
}

type StorageBase struct {
	PrimaryType string `json:"primary_type"`

	PrimaryName string `json:"-"`
	PrimaryDC   string `json:"-"`
	PrimaryId   string `json:"primary_id"`   // fcp or iscsi - dr_domain_id
	PrimaryAddr string `json:"primary_addr"` // nfs, iscsi
	PrimaryPath string `json:"primary_path"` // nfs,
	PrimaryPort string `json:"primary_port"` // iscsi

	SecondaryName string `json:"-"`
	SecondaryDC   string `json:"-"`
	SecondaryType string `json:"secondary_type"`
	SecondaryId   string `json:"secondary_id"`   // fcp or iscsi - verify dr_domain_id
	SecondaryAddr string `json:"secondary_addr"` // nfs, iscsi
	SecondaryPath string `json:"secondary_path"` // nfs
	SecondaryPort string `json:"secondary_port"` // iscsi
}

// StorageMap for nfs
// {
// 	'primary_type': 'nfs', 'primary_addr': '192.168.122.210', 'primary_path': '/nfs_dom',
// 	'secondary_type': 'nfs', 'secondary_addr': '192.168.122.210', 'secondary_path': '/nfs_dom_replica'
// }

// StorageMap for fcp
// {
// 	'primary_type': 'fcp', 'primary_id': '0abc45defc',
// 	'secondary_type': 'fcp', 'secondary_id': '0abc45defc',
// }

// StorageMap for iscsi
// {
// 	'primary_type': 'iscsi', 'primary_id': 'bcca8438-810f-4932-bf25-d874babd97b1', 'primary_addr': '192.168.1.101', 'primary_port': '3260',
// 	'secondary_type': 'iscsi', 'secondary_id': 'bcca8438-810f-4932-bf25-d874babd97b1', 'secondary_addr': '192.168.2.101', 'secondary_port': '3260',
//  'targets': {
//     'iqn.2006-01.com.openfiler:olvm-data1': 'iqn.2006-02.com.openfiler:olvm-data1-2',
//     'iqn.2006-01.com.openfiler:olvm-data2': 'iqn.2006-02.com.openfiler:olvm-data2-2'
//   }
// }

type StorageMap struct {
	StorageBase

	Targets map[string]string `json:"targets"` // iscsi

	Found bool `json:"-"` // found during remap flag
}

// ["iqn.2006-01.com.openfiler:olvm-data1", "iqn.2006-01.com.openfiler:olvm-data2"]
func splitYamlStringList(v string) (vals []string, err error) {
	if strings.HasPrefix(v, `["`) && strings.HasSuffix(v, `"]`) {
		v = v[2 : len(v)-2]
		vals = strings.Split(v, `", "`)
		return
	}
	return nil, errors.New("unclosed list: " + v)
}

func joinYamlStringList(vals []string) string {
	if len(vals) == 0 {
		return "[]"
	}
	return `["` + strings.Join(vals, `", "`) + `"]`
}

func remapStorageMap(values []*ayaml.Node, storageDomains []StorageMap) (success bool, msg string, errs []error) {
	m := make(map[string]string)
	for _, val := range values {
		if s, ok := val.Value.(string); ok {
			m[val.Key] = s
		}
	}

	domainType := m["dr_domain_type"]
	switch domainType {
	case "nfs":
		m["dr_primary_path"] = strings.TrimRight(m["dr_primary_path"], "/")
		m["dr_secondary_path"] = strings.TrimRight(m["dr_secondary_path"], "/")
		for n, domain := range storageDomains {
			if domainType != domain.PrimaryType || domain.PrimaryType != domain.SecondaryType {
				continue
			}
			if m["dr_primary_address"] != domain.PrimaryAddr || m["dr_primary_path"] != domain.PrimaryPath {
				continue
			}

			m["dr_secondary_address"] = domain.SecondaryAddr
			m["dr_secondary_path"] = domain.SecondaryPath

			storageDomains[n].Found = true

			key := domainType + "://" + m["dr_secondary_address"] + ":" + m["dr_secondary_path"]
			success = true
			msg = "storage " + m["dr_primary_name"] + " remapped with name " + m["dr_secondary_name"] + " as " + key

			break
		}
	case StorageFCP:
		for n, domain := range storageDomains {
			if domainType != domain.PrimaryType || domain.PrimaryType != domain.SecondaryType {
				continue
			}
			if m["dr_domain_id"] != domain.PrimaryId {
				continue
			}

			storageDomains[n].Found = true

			key := domainType + "://" + m["dr_domain_id"]
			success = true
			msg = "storage " + m["dr_primary_name"] + " remapped with name " + m["dr_secondary_name"] + " as " + key

			break
		}
	case StorageISCSI:
		for n, domain := range storageDomains {
			if domainType != domain.PrimaryType || domain.PrimaryType != domain.SecondaryType {
				continue
			}
			if m["dr_domain_id"] != domain.PrimaryId {
				continue
			}
			if m["dr_primary_address"] != domain.PrimaryAddr || m["dr_primary_port"] != domain.PrimaryPort {
				continue
			}

			mPrimaryTargets, err := splitYamlStringList(m["dr_primary_target"])
			if err != nil {
				errs = append(errs, err)
			}

			primaryTargets := make([]string, 0, len(mPrimaryTargets))
			secondaryTargets := make([]string, 0, len(mPrimaryTargets))
			for i := 0; i < len(mPrimaryTargets); i++ {
				if s, ok := domain.Targets[mPrimaryTargets[i]]; ok {
					primaryTargets = append(primaryTargets, mPrimaryTargets[i])
					secondaryTargets = append(secondaryTargets, s)
				}
			}
			if len(primaryTargets) == 0 {
				continue
			}

			m["dr_primary_target"] = joinYamlStringList(primaryTargets)
			m["dr_secondary_target"] = joinYamlStringList(secondaryTargets)

			m["dr_secondary_address"] = domain.SecondaryAddr
			m["dr_secondary_port"] = domain.SecondaryPort

			storageDomains[n].Found = true

			key := domainType + "://" + m["dr_domain_id"]
			success = true
			targets := strings.Join(secondaryTargets, ",")
			msg = "storage " + m["dr_primary_name"] + " remapped with name " + m["dr_secondary_name"] + " as " + key + ":" + targets

			break
		}
	}

	if success {
		// return remapped values back
		for _, val := range values {
			if _, ok := val.Value.(string); ok {
				if v, ok := m[val.Key]; ok {
					val.Value = v
				}
			}
		}
	}

	return
}

func RemapStorages(node *ayaml.Node, storageDomains []StorageMap) (success bool, msgs []string, errs []error) {
	if nodes, ok := node.Value.([]*ayaml.Node); ok {
		for _, node := range nodes {
			if node.Key == "dr_import_storages" {
				success = true
				storages := node.Value.([]*ayaml.Node)
				var newStorages []*ayaml.Node
				for _, v := range storages {
					if v.Value != nil {
						var (
							ok   bool
							msg  string
							err1 []error
						)
						values := v.Value.([]*ayaml.Node)
						if ok, msg, err1 = remapStorageMap(values, storageDomains); ok {
							newStorages = append(newStorages, v)
						}
						if msg != "" {
							msgs = append(msgs, msg)
						}
						if len(err1) > 0 {
							errs = append(errs, err1...)
						}
					}
				}
				node.Value = newStorages
				if len(newStorages) == 0 {
					success = false
				}
				break
			}
		}
	} else {
		errs = append(errs, ErrAnsibleFileNoStorages)
	}
	return
}

// GenerateVars is OVirt engines API address/credentials
type GenerateVars struct {
	PrimaryUrl        string            `json:"site_primary_url"`
	PrimaryUsername   string            `json:"site_primary_username"`
	PrimaryPassword   string            `json:"site_primary_password"`
	SecondaryUrl      string            `json:"site_secondary_url"`
	SecondaryUsername string            `json:"site_secondary_username"`
	SecondaryPassword string            `json:"site_secondary_password"`
	StorageDomains    []StorageMap      `json:"storage_domains"`
	Rewrite           map[string]string `json:"rewrite"` // ~ for delete
}

func (g GenerateVars) Generate(name, dir string) (storages []string, out string, err error) {
	ansiblePath, err := exec.LookPath("ansible-playbook")
	if err != nil {
		err = ErrAnsibleNotFound
		return
	}

	if !validateName(name) {
		err = ErrNameInvalid
		return
	}
	template := path.Join(dir, "template")
	dir = path.Join(dir, name)
	if utils.DirExists(dir) {
		err = ErrDirAlreadyExist
		return
	}
	var warnings []error

	ansibleGeneratePlaybook := path.Join(dir, ansibleGeneratePlaybook)
	ansibleFailoverPlaybook := path.Join(dir, ansibleFailoverPlaybook)
	ansibleFailbackPlaybook := path.Join(dir, ansibleFailbackPlaybook)
	ansibleVarFile := path.Join(dir, ansibleDrVarsFile)
	ansibleVarFileTpl := ansibleVarFile + ".tpl"
	primaryCaFile := path.Join(dir, "primary.ca")
	secondaryCaFile := path.Join(dir, "secondary.ca")

	if !lock.TryLock() {
		return nil, "", ErrInProgress
	}

	flock := fslock.New(dir + ".lock")
	if err = flock.TryLock(); err != nil {
		lock.Unlock()
		return
	}

	wg.Add(1)

	go func() {
		defer func() {
			lock.Unlock()
			_ = flock.Unlock()
			wg.Done()
		}()

		if err = cp.Copy(template, dir); err != nil {
			return
		}

		if err = saveCaFile(g.PrimaryUrl, primaryCaFile); err != nil {
			return
		}
		if err = validateOvirtCon(g.PrimaryUrl, false, primaryCaFile, g.PrimaryUsername, g.PrimaryPassword); err != nil {
			return
		}

		if err = saveCaFile(g.SecondaryUrl, secondaryCaFile); err != nil {
			return
		}
		if err = validateOvirtCon(g.SecondaryUrl, false, secondaryCaFile, g.SecondaryUsername, g.SecondaryPassword); err != nil {
			return
		}

		if err = g.writeAnsiblePwdDile(path.Join(dir, ansibleDrPwdFile)); err != nil {
			return
		}

		extraVars := "site=" + g.PrimaryUrl + " username=" + g.PrimaryUsername + " password=" + g.PrimaryPassword +
			" ca=" + primaryCaFile + " var_file=" + ansibleVarFileTpl

		// TODO: reduce verbose ?
		if out, err = utils.ExecCmd(dir+"/generate.log", time.Minute*10, ansiblePath, ansibleGeneratePlaybook, "-t", ansibleDrTag, "-e", extraVars, "-vvvvv"); err == nil {
			if utils.FileExists(ansibleVarFileTpl) {
				if err = g.writeAnsibleFailbackFile(ansibleFailoverPlaybook, ansibleFailbackPlaybook); err == nil {
					storages, warnings, err = g.writeAnsibleVarsFile(ansibleVarFileTpl, ansibleVarFile)
				}
			} else {
				err = ErrVarFileNotExist
			}
		}
	}()

	wg.Wait()

	var buf strings.Builder

	if len(storages) > 0 {
		buf.WriteString("STORAGES:\n")
		for _, storage := range storages {
			buf.WriteString(storage)
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
		buf.WriteString(out)
	}

	if len(warnings) > 0 {
		buf.WriteString("STORAGES WARNINGS:\n")
		for _, warn := range warnings {
			buf.WriteString(warn.Error())
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
		buf.WriteString(out)
	}

	out = buf.String()

	return
}

func (g GenerateVars) Validate() error {
	var errs Errors

	if g.PrimaryUrl == "" {
		errs = append(errs, "site_primary_url is empty")
	} else if !strings.HasPrefix(g.PrimaryUrl, "https://") {
		errs = append(errs, "site_primary_url is invalid")
	}
	if g.PrimaryUsername == "" {
		errs = append(errs, "site_primary_username is empty")
	}
	if g.PrimaryPassword == "" {
		errs = append(errs, "site_primary_password is empty")
	}

	if g.SecondaryUrl == "" {
		errs = append(errs, "site_secondary_url is empty")
	} else if !strings.HasPrefix(g.SecondaryUrl, "https://") {
		errs = append(errs, "site_secondary_url is invalid")
	}
	if g.SecondaryUsername == "" {
		errs = append(errs, "site_secondary_username is empty")
	}
	if g.SecondaryPassword == "" {
		errs = append(errs, "site_secondary_password is empty")
	}

	for i, s := range g.StorageDomains {
		if s.PrimaryType == "" {
			errs = append(errs, "primary_type["+strconv.Itoa(i)+"] is empty")
		}
		if s.SecondaryType != s.PrimaryType {
			errs = append(errs, "secondary_type["+strconv.Itoa(i)+"] mismatch")
		}
		switch s.PrimaryType {
		case StorageNFS:
			if s.PrimaryAddr == "" {
				errs = append(errs, "primary_addr["+strconv.Itoa(i)+"] is empty")
			}
			if s.PrimaryPath == "" {
				errs = append(errs, "primary_path["+strconv.Itoa(i)+"] is empty")
			}

			if s.SecondaryAddr == "" {
				errs = append(errs, "secondary_addr["+strconv.Itoa(i)+"] is empty")
			}
			if s.SecondaryPath == "" {
				errs = append(errs, "secondary_path["+strconv.Itoa(i)+"] is empty")
			}
		case StorageFCP:
			if s.PrimaryId == "" {
				errs = append(errs, "primary_id["+strconv.Itoa(i)+"] is empty")
			}

			if s.SecondaryId == "" {
				errs = append(errs, "secondary_id["+strconv.Itoa(i)+"] is empty")
			}
		case StorageISCSI:
			if s.PrimaryId == "" {
				errs = append(errs, "primary_id["+strconv.Itoa(i)+"] is empty")
			}
			if s.PrimaryAddr == "" {
				errs = append(errs, "primary_addr["+strconv.Itoa(i)+"] is empty")
			}
			if s.PrimaryPort == "" {
				errs = append(errs, "primary_port["+strconv.Itoa(i)+"] is empty")
			}

			if s.SecondaryId == "" {
				errs = append(errs, "secondary_id["+strconv.Itoa(i)+"] is empty")
			}
			if s.SecondaryAddr == "" {
				errs = append(errs, "secondary_addr["+strconv.Itoa(i)+"] is empty")
			}
			if s.SecondaryPort == "" {
				errs = append(errs, "secondary_port["+strconv.Itoa(i)+"] is empty")
			}

			if len(s.Targets) == 0 {
				errs = append(errs, "targets["+strconv.Itoa(i)+"] is empty")
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (g GenerateVars) writeAnsiblePwdDile(pwdFile string) error {
	f, err := os.OpenFile(pwdFile, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.Grow(128)
	// TODO: encrypt with ansible vault
	buf.WriteString("dr_sites_primary_password: ")
	buf.WriteString(g.PrimaryPassword)
	buf.WriteString("\ndr_sites_secondary_password: ")
	buf.WriteString(g.SecondaryPassword)
	_, err = f.Write(buf.Bytes())
	return err
}

func (g GenerateVars) rewrite(node *ayaml.Node) {
	if nodes, ok := node.Value.([]*ayaml.Node); ok {
		for _, node := range nodes {
			if node.Type == ayaml.NodeString {
				switch node.Key {
				case "dr_sites_secondary_url":
					node.Value = g.SecondaryUrl
				case "dr_sites_secondary_username":
					node.Value = g.SecondaryUsername
				case "dr_sites_secondary_ca_file":
					node.Value = strings.Replace(node.Value.(string), "/primary.ca", "/secondary.ca", 1)
				}
			}
		}
	}
}

func (g GenerateVars) writeAnsibleVarsFile(template, varFile string) (storages []string, remapWarnings []error, err error) {
	var (
		in, out *os.File
		nodes   *ayaml.Node
	)
	if in, err = os.Open(template); err != nil {
		return
	}
	defer in.Close()

	g.StorageDomains = StripStorageDomains(g.StorageDomains)

	if nodes, err = ayaml.Decode(in); err != nil {
		return
	}

	ok, msg, rErr := RemapStorages(nodes, g.StorageDomains)
	if len(rErr) > 0 {
		remapWarnings = append(remapWarnings, rErr...)
	}
	if ok {
		storages = msg
	}

	g.rewrite(nodes)

	ayaml.Rewrite(nodes, g.Rewrite)

	if out, err = os.OpenFile(varFile, os.O_RDWR|os.O_CREATE, 0644); err != nil {
		return
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	if err = nodes.Write(w); err != nil {
		return
	}

	if !ok {
		err = ErrStorageRemapEmptyResult
	}

	for _, domain := range g.StorageDomains {
		if !domain.Found {
			key := domain.PrimaryType + "://" + domain.PrimaryAddr + ":" + domain.PrimaryPath
			remapWarnings = append(remapWarnings, errors.New("storage map "+key+" not used"))
		}
	}

	return
}

func (g GenerateVars) writeAnsibleFailbackFile(failover, failback string) (err error) {
	var in, out *os.File
	in, err = os.Open(failover)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err = os.OpenFile(failback, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := bufio.NewWriter(out)

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "     dr_source_map: ") {
			if _, err = writer.WriteString("     dr_source_map: secondary\n"); err != nil {
				return
			}
		} else if strings.HasPrefix(s, "     dr_target_host: ") {
			if _, err = writer.WriteString("     dr_target_host: primary\n"); err != nil {
				return
			}
		} else {
			if _, err = writer.WriteString(s); err != nil {
				return
			}
			if err = writer.WriteByte('\n'); err != nil {
				return
			}
		}
	}
	if err = writer.Flush(); err != nil {
		return
	}

	return scanner.Err()
}
