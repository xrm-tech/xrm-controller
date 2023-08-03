package ovirt

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/juju/fslock"
	cp "github.com/otiai10/copy"
	ovirtsdk4 "github.com/ovirt/go-ovirt"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	ErrImportStorageItem       = errors.New("dr_import_storages item parse error")
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

func writeStringLn(w *bufio.Writer, s string) (err error) {
	if _, err = w.WriteString(s); err != nil {
		return
	}
	err = w.WriteByte('\n')
	return
}

func writeKVLn(w *bufio.Writer, k, v string) (err error) {
	if _, err = w.WriteString(k); err != nil {
		return
	}
	if _, err = w.WriteString(": "); err != nil {
		return
	}
	if _, err = w.WriteString(v); err != nil {
		return
	}
	err = w.WriteByte('\n')
	return
}

var (
	commentRe = regexp.MustCompile(`^ *#`)
)

func splitKV(s string, uncomment bool) (k, v string, ok bool) {
	if commentRe.MatchString(s) {
		return
	}
	k, v, ok = strings.Cut(s, ":")
	if ok {
		k = strings.TrimRight(k, " ")
		v = strings.TrimLeft(v, " ")
		if uncomment && strings.HasPrefix(v, "#") {
			v = strings.TrimLeft(v, "#")
			v = strings.TrimLeft(v, " ")
		}
		if v == "" {
			ok = false
		}
	}
	return
}

func startBytes(s string, r rune) (n int) {
	l := utf8.RuneLen(r)
	for _, c := range s {
		if c == r {
			n += l
		} else {
			break
		}
	}
	return
}

func StripStorageDomains(s []Storage) []Storage {
	pos := 0
	exist := make(map[string]bool)
	for i := 0; i < len(s); i++ {
		if s[i].PrimaryType == "nfs" {
			s[i].PrimaryPath = strings.TrimRight(s[i].PrimaryPath, "/")
			s[i].SecondaryPath = strings.TrimRight(s[i].SecondaryPath, "/")
		}
		// cleanup from dulpicates
		key := s[i].PrimaryType + "://" + s[i].PrimaryAddr + ":" + s[i].PrimaryPath
		if _, ok := exist[key]; !ok {
			s[pos] = s[i]
			pos++
			exist[key] = true
		}

	}
	return s[:pos]
}

type KV struct {
	K, V string
}

type Storage struct {
	PrimaryType   string `json:"primary_type"`
	PrimaryName   string `json:"-"`
	PrimaryDC     string `json:"-"`
	PrimaryPath   string `json:"primary_path"` // nfs
	PrimaryAddr   string `json:"primary_addr"` // nfs, for fcp - dr_domain_id
	SecondaryType string `json:"secondary_type"`
	SecondaryName string `json:"-"`
	SecondaryDC   string `json:"-"`
	SecondaryPath string `json:"secondary_path"` // nfs
	SecondaryAddr string `json:"secondary_addr"` // nfs, for fcp - verify dr_domain_id
	Params        []KV   `json:"-"`
	Found         bool   `json:"-"`
}

// {
// 	'primary_type': 'nfs', 'primary_addr': '192.168.122.210', 'primary_path': '/nfs_dom',
// 	'secondary_type': 'nfs', 'secondary_addr': '192.168.122.210', 'secondary_path': '/nfs_dom_replica'
// }

func (m *Storage) Reset() {
	m.PrimaryType = ""
	m.PrimaryDC = ""
	m.PrimaryName = ""
	m.PrimaryPath = ""
	m.PrimaryAddr = ""
	m.SecondaryType = ""
	m.SecondaryDC = ""
	m.SecondaryName = ""
	m.SecondaryPath = ""
	m.SecondaryAddr = ""

	if len(m.Params) > 0 {
		m.Params = m.Params[:0]
	}
}

func (m *Storage) Set(s string) {
	if strings.HasPrefix(s, "#") {
		return
	}
	k, v, ok := strings.Cut(s, ": ")
	if ok {
		v = strings.TrimPrefix(v, " ")
		v = strings.TrimPrefix(v, "# ")
		switch k {
		case "dr_domain_type":
			m.PrimaryType = v
			m.SecondaryType = v
		case "dr_primary_name":
			m.PrimaryName = v
		case "dr_primary_dc_name":
			m.PrimaryDC = v
		case "dr_primary_path": // nfs
			m.PrimaryPath = v
		case "dr_primary_address": // nfs
			m.PrimaryAddr = v
		case "dr_secondary_name":
			m.SecondaryName = v
		case "dr_secondary_dc_name":
			m.SecondaryDC = v
		case "dr_secondary_address": // nfs
			m.SecondaryAddr = v
		case "dr_secondary_path": // nfs
			m.SecondaryPath = v
		case "dr_domain_id": // fcp
			m.PrimaryAddr = v
			m.SecondaryAddr = v
		}
		m.Params = append(m.Params, KV{K: k, V: v})
	}
}

func (m *Storage) Remap(storageDomains []Storage) (ok bool, msgs error) {
	if m.PrimaryType == "nfs" {
		if m.PrimaryAddr == "" {
			return false, errors.New("storage for " + m.PrimaryName + " without address")
		}
		if m.PrimaryPath == "" {
			return false, errors.New("storage for " + m.PrimaryName + " without path")
		}

		m.PrimaryPath = strings.TrimRight(m.PrimaryPath, "/")
		m.SecondaryPath = strings.TrimRight(m.SecondaryPath, "/")
		for n, domain := range storageDomains {
			if m.PrimaryType != domain.PrimaryType || domain.PrimaryType != domain.SecondaryType {
				continue
			}
			if domain.PrimaryName != "" && m.PrimaryName != domain.PrimaryName {
				continue
			}
			if m.PrimaryAddr != domain.PrimaryAddr || m.PrimaryPath != domain.PrimaryPath {
				continue
			}

			if domain.SecondaryName != "" {
				m.SecondaryName = domain.SecondaryName
			} else if domain.PrimaryName != "" {
				m.SecondaryName = domain.PrimaryName
			}
			if domain.SecondaryDC != "" {
				m.SecondaryDC = domain.SecondaryDC
			} else if domain.PrimaryDC != "" {
				m.SecondaryDC = m.PrimaryDC
			}
			if domain.SecondaryAddr != "" {
				m.SecondaryAddr = domain.SecondaryAddr
			}
			if domain.SecondaryPath != "" {
				m.SecondaryPath = domain.SecondaryPath
			}
			storageDomains[n].Found = true
			var key string
			if m.SecondaryPath == "" {
				key = m.SecondaryType + "://" + m.SecondaryAddr
			} else {
				key = m.SecondaryType + "://" + m.SecondaryAddr + ":" + m.SecondaryPath
			}
			return true, errors.New("storage " + m.PrimaryName + " remapped with name " + m.SecondaryName + " as " + key)
		}
	} else if m.PrimaryType == "fcp" {
		if m.PrimaryAddr == "" {
			return false, errors.New("storage for " + m.PrimaryName + " without address")
		}
		if m.PrimaryPath != "" {
			return false, errors.New("storage for " + m.PrimaryName + " can't contain path")
		}
		if m.PrimaryAddr != m.SecondaryAddr {
			return false, errors.New("storage for " + m.PrimaryName + " with secondary address mismatch")
		}

		for n, domain := range storageDomains {
			if m.PrimaryType != domain.PrimaryType || (domain.PrimaryType != domain.SecondaryType && domain.SecondaryType != "") {
				continue
			}
			if domain.PrimaryName != "" && m.PrimaryName != domain.PrimaryName {
				continue
			}

			if domain.SecondaryName != "" {
				m.SecondaryName = domain.SecondaryName
			} else if domain.PrimaryName != "" {
				m.SecondaryName = domain.PrimaryName
			}
			if domain.SecondaryDC != "" {
				m.SecondaryDC = domain.SecondaryDC
			} else if domain.PrimaryDC != "" {
				m.SecondaryDC = m.PrimaryDC
			}

			storageDomains[n].Found = true
			var key string
			if m.SecondaryPath == "" {
				key = m.SecondaryType + "://" + m.SecondaryAddr
			} else {
				key = m.SecondaryType + "://" + m.SecondaryAddr + ":" + m.SecondaryPath
			}

			return true, errors.New("storage " + m.PrimaryName + " remapped with name " + m.SecondaryName + " as " + key)
		}
	}

	return false, errors.New("storage map for " + m.PrimaryName + " not found")
}

func (m *Storage) remap(name, originValue string) string {
	switch name {
	case "dr_domain_type":
		return m.PrimaryType
	case "dr_primary_name":
		return m.PrimaryName
	case "dr_primary_dc_name":
		return m.PrimaryDC
	case "dr_primary_path": // nfs
		return m.PrimaryPath
	case "dr_primary_address": // nfs
		return m.PrimaryAddr
	case "dr_secondary_name":
		return m.SecondaryName
	case "dr_secondary_dc_name":
		return m.SecondaryDC
	case "dr_secondary_address": // nfs
		return m.SecondaryAddr
	case "dr_secondary_path": // nfs
		return m.SecondaryPath
	case "dr_domain_id": // fcp
		return m.PrimaryAddr
	default:
		return originValue
	}
}

func (m *Storage) WriteAnsibleMap(w *bufio.Writer) error {

	for i := 0; i < len(m.Params); i++ {
		var prefixedKey string
		if i == 0 {
			prefixedKey = "- " + m.Params[i].K
		} else {
			prefixedKey = "  " + m.Params[i].K
		}

		if err := writeKVLn(w, prefixedKey, m.remap(m.Params[i].K, m.Params[i].V)); err != nil {
			return err
		}
	}

	return nil
}

func (m *Storage) WriteString(buf *strings.Builder) {
	buf.WriteByte('{')

	for i := 0; i < len(m.Params); i++ {
		if i == 0 {
			_ = buf.WriteByte(' ')
		} else {
			_, _ = buf.WriteString(", ")
		}
		_, _ = buf.WriteString(m.Params[i].K)
		_, _ = buf.WriteString(": \"")
		_, _ = buf.WriteString(m.remap(m.Params[i].K, m.Params[i].V))
		_, _ = buf.WriteString("\"")
	}

	buf.WriteByte('}')
}

// GenerateVars is OVirt engines API address/credentials
type GenerateVars struct {
	PrimaryUrl        string    `json:"site_primary_url"`
	PrimaryUsername   string    `json:"site_primary_username"`
	PrimaryPassword   string    `json:"site_primary_password"`
	SecondaryUrl      string    `json:"site_secondary_url"`
	SecondaryUsername string    `json:"site_secondary_username"`
	SecondaryPassword string    `json:"site_secondary_password"`
	StorageDomains    []Storage `json:"storage_domains"`
}

func (g GenerateVars) Generate(name, dir string) (storages string, out string, err error) {
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
		return "", "", ErrInProgress
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

	if len(warnings) > 0 {
		var buf strings.Builder
		buf.WriteString("STORAGES MESSAGES AND WARNINGS:\n")
		for _, warn := range warnings {
			buf.WriteString(warn.Error())
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
		buf.WriteString(out)
		out = buf.String()
	}

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
		if s.PrimaryPath == "" {
			errs = append(errs, "primary_path["+strconv.Itoa(i)+"] is empty")
		}
		if s.PrimaryAddr == "" {
			errs = append(errs, "primary_addr["+strconv.Itoa(i)+"] is empty")
		}

		if s.SecondaryType == "" {
			errs = append(errs, "secondary_type["+strconv.Itoa(i)+"] is empty")
		}
		if s.SecondaryPath == "" {
			errs = append(errs, "secondary_path["+strconv.Itoa(i)+"] is empty")
		}
		if s.SecondaryAddr == "" {
			errs = append(errs, "secondary_addr["+strconv.Itoa(i)+"] is empty")
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

type importState int8

const (
	importNone importState = iota
	importStorage
)

func (g GenerateVars) writeAnsibleVarsFile(template, varFile string) (storages string, remapWarnings []error, err error) {
	var in, out *os.File
	in, err = os.Open(template)
	if err != nil {
		return
	}
	defer in.Close()

	out, err = os.OpenFile(varFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer out.Close()
	writer := bufio.NewWriter(out)

	var (
		importPhase importState
		storage     Storage
	)

	g.StorageDomains = StripStorageDomains(g.StorageDomains)

	storagesSlice := make([]Storage, 0, 4)

	indent := 0
	hasStorages := false
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		s := scanner.Text()
		switch importPhase {
		case importStorage:
			if strings.HasPrefix(s, "- ") {
				if storage.PrimaryType != "" {
					// flush map
					ok, rErr := storage.Remap(g.StorageDomains)
					storagesSlice = append(storagesSlice, storage)
					if rErr != nil {
						remapWarnings = append(remapWarnings, rErr)
					}
					if ok {
						hasStorages = true
						if err = storage.WriteAnsibleMap(writer); err != nil {
							return
						}
					}
				}
				storage.Reset()

				indent = startBytes(s[1:], ' ') + 1
				if indent != 2 {
					err = ErrImportStorageItem
					return
				}
				storage.Set(s[indent:])
			} else if strings.HasPrefix(s, "  ") {
				storage.Set(s[2:])
			} else {
				// break map
				if storage.PrimaryType != "" {
					ok, rErr := storage.Remap(g.StorageDomains)
					storagesSlice = append(storagesSlice, storage)
					if rErr != nil {
						remapWarnings = append(remapWarnings, rErr)
					}
					if ok {
						hasStorages = true
						if err = storage.WriteAnsibleMap(writer); err != nil {
							return
						}
					}
				}

				storage.Reset()

				if s == "" {
					if err = writeStringLn(writer, s); err != nil {
						return
					}
				} else {
					importPhase = importNone

					k, v, ok := splitKV(s, true)
					if ok {
						if err = writeKVLn(writer, k, v); err != nil {
							return
						}
					} else if err = writeStringLn(writer, s); err != nil {
						return
					}
				}
			}
		default:
			if strings.HasPrefix(s, "dr_sites_secondary_url: ") {
				if err = writeKVLn(writer, "dr_sites_secondary_url", g.SecondaryUrl); err != nil {
					return
				}
			} else if strings.HasPrefix(s, "dr_sites_secondary_username: ") {
				if err = writeKVLn(writer, "dr_sites_secondary_username", g.SecondaryUsername); err != nil {
					return
				}
			} else if strings.HasPrefix(s, "dr_sites_secondary_ca_file: ") {
				k, v, _ := splitKV(s, true)
				if err = writeKVLn(writer, k, strings.Replace(v, "primary.ca", "secondary.ca", 1)); err != nil {
					return
				}
			} else if s == "dr_import_storages:" {
				if err = writeStringLn(writer, s); err != nil {
					return
				}

				importPhase = importStorage
				storage.Reset()
			} else {
				k, v, ok := splitKV(s, true)
				if ok {
					if err = writeKVLn(writer, k, v); err != nil {
						return
					}
				} else if err = writeStringLn(writer, s); err != nil {
					return
				}
			}
		}
	}

	if importPhase == importStorage && storage.PrimaryType != "" {
		ok, rErr := storage.Remap(g.StorageDomains)
		storagesSlice = append(storagesSlice, storage)
		if rErr != nil {
			remapWarnings = append(remapWarnings, rErr)
		}
		if ok {
			hasStorages = true
			if err = storage.WriteAnsibleMap(writer); err != nil {
				return
			}
		}
	}

	if err = writer.Flush(); err != nil {
		return
	}

	err = scanner.Err()
	if err != nil {
		return
	}
	if !hasStorages {
		err = ErrStorageRemapEmptyResult
	}

	for i, domain := range g.StorageDomains {
		if domain.Found {
			g.StorageDomains[i].Found = false
		} else {
			key := domain.PrimaryType + "://" + domain.PrimaryAddr + ":" + domain.PrimaryPath
			remapWarnings = append(remapWarnings, errors.New("storage map "+key+" not used"))
		}
	}

	if len(storagesSlice) > 0 {
		var buf strings.Builder
		buf.WriteString("[\n")
		for i, s := range storagesSlice {
			if i > 0 {
				buf.WriteString(",\n")
			}
			buf.WriteString("  ")
			s.WriteString(&buf)
		}
		buf.WriteString("\n]")
		storages = buf.String()
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
