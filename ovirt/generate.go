package ovirt

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/juju/fslock"
	cp "github.com/otiai10/copy"
	ovirtsdk4 "github.com/ovirt/go-ovirt"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	ErrImportStorageItem = errors.New("dr_import_storages item parse error")

	ErrTemplateDirNotExist = errors.New("ovirt template dir not exist")
	ErrDirAlreadyExist     = errors.New("dir already exist")
	ErrVarFileNotExist     = errors.New("var file not exist")
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

func writeEntryLn(w *bufio.Writer, k, v string) error {
	if _, err := w.WriteString(k); err != nil {
		return err
	}
	if _, err := w.WriteString(v); err != nil {
		return err
	}
	return w.WriteByte('\n')
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

type Storage struct {
	StorageType   string   `json:"primary_type" validate:"required"`
	PrimaryName   string   `json:"-"`
	PrimaryDC     string   `json:"-"`
	PrimaryPath   string   `json:"primary_path" validate:"required"`
	PrimaryAddr   string   `json:"primary_addr" validate:"required"`
	SecondaryName string   `json:"-"`
	SecondaryDC   string   `json:"-"`
	SecondaryPath string   `json:"secondary_path" validate:"required"`
	SecondaryAddr string   `json:"secondary_addr" validate:"required"`
	Additional    []string `json:"-"`
}

// {
// 	'primary_type': 'nfs', 'primary_addr': '192.168.122.210', 'primary_path': '/nfs_dom',
// 	'secondary_type': 'nfs', 'secondary_addr': '192.168.122.210', 'secondary_path': '/nfs_dom_replica'
// }

func (m *Storage) Reset() {
	m.StorageType = ""
	m.PrimaryDC = ""
	m.PrimaryName = ""
	m.PrimaryPath = ""
	m.PrimaryAddr = ""
	m.SecondaryDC = ""
	m.SecondaryName = ""
	m.SecondaryPath = ""
	m.SecondaryAddr = ""
	m.Additional = m.Additional[:0]
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
			m.StorageType = v
		case "dr_primary_name":
			m.PrimaryName = v
		case "dr_primary_dc_name":
			m.PrimaryDC = v
		case "dr_primary_path":
			m.PrimaryPath = v
		case "dr_primary_address":
			m.PrimaryAddr = v
		case "dr_secondary_name":
			m.SecondaryName = v
		case "dr_secondary_dc_name":
			m.SecondaryDC = v
		case "dr_secondary_address":
			m.SecondaryAddr = v
		case "dr_secondary_path":
			m.SecondaryPath = v
		default:
			m.Additional = append(m.Additional, k+": "+v)
		}
	} else {
		m.Additional = append(m.Additional, s)
	}
}

func (m *Storage) Remap(storageDomains []Storage) (bool, error) {
	for _, domain := range storageDomains {
		if m.StorageType != domain.StorageType {
			continue
		}
		if domain.PrimaryName != "" && m.PrimaryName != domain.PrimaryName {
			continue
		}
		if domain.PrimaryAddr != "" && m.PrimaryAddr != domain.PrimaryAddr {
			continue
		}
		if domain.PrimaryPath != "" && m.PrimaryPath != domain.PrimaryPath {
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
		return true, nil
	}
	return true, errors.New("storage map for " + m.PrimaryName + " not found")
}

func (m *Storage) Write(w *bufio.Writer) error {
	if err := writeEntryLn(w, "- dr_domain_type: ", m.StorageType); err != nil {
		return err
	}

	if err := writeEntryLn(w, "  dr_primary_name: ", m.PrimaryName); err != nil {
		return err
	}
	if err := writeEntryLn(w, "  dr_primary_dc_name: ", m.PrimaryDC); err != nil {
		return err
	}
	if err := writeEntryLn(w, "  dr_primary_path: ", m.PrimaryPath); err != nil {
		return err
	}
	if err := writeEntryLn(w, "  dr_primary_address: ", m.PrimaryAddr); err != nil {
		return err
	}

	if err := writeEntryLn(w, "  dr_secondary_name: ", m.SecondaryName); err != nil {
		return err
	}
	if err := writeEntryLn(w, "  dr_secondary_dc_name: ", m.SecondaryDC); err != nil {
		return err
	}
	if err := writeEntryLn(w, "  dr_secondary_path: ", m.SecondaryPath); err != nil {
		return err
	}
	if err := writeEntryLn(w, "  dr_secondary_address: ", m.SecondaryAddr); err != nil {
		return err
	}

	for _, a := range m.Additional {
		if err := writeEntryLn(w, "  ", a); err != nil {
			return err
		}
	}

	return nil
}

// GenerateVars is OVirt engines API address/credentials
type GenerateVars struct {
	PrimaryUrl        string    `json:"site_primary_url" validate:"required,startswith=https://"`
	PrimaryUsername   string    `json:"site_primary_username" validate:"required"`
	PrimaryPassword   string    `json:"site_primary_password" validate:"required"`
	SecondaryUrl      string    `json:"site_secondary_url" validate:"required,startswith=https://"`
	SecondaryUsername string    `json:"site_secondary_username" validate:"required"`
	SecondaryPassword string    `json:"site_secondary_password" validate:"required"`
	StorageDomains    []Storage `json:"storage_domains" validate:"required"`
}

func (g GenerateVars) Generate(name, dir string) (out string, err error) {
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
		return "", ErrInProgress
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
				err = g.writeAnsibleFailbackFile(ansibleFailoverPlaybook, ansibleFailbackPlaybook)
				if err == nil {
					warnings, err = g.writeAnsibleVarsFile(ansibleVarFileTpl, ansibleVarFile)
				}
			} else {
				err = ErrVarFileNotExist
			}
		}
	}()

	wg.Wait()

	if len(warnings) > 0 {
		var buf strings.Builder
		buf.WriteString(out)
		buf.WriteString("\nWARNINGS:\n")
		for _, warn := range warnings {
			buf.WriteString(warn.Error())
			buf.WriteByte('\n')
		}
	}

	return
}

func (g GenerateVars) Validate() error {
	return Validate.Struct(&g)
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

type importStorageState int8

const (
	importStorageNone = iota
	importStorageWant
	importStorageStarted
)

func (g GenerateVars) writeAnsibleVarsFile(template, varFile string) (remapWarnings []error, err error) {
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
		importStorage importStorageState
		storage       Storage
	)
	indent := 0
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		s := scanner.Text()
		// if blankRe.MatchString(s) {
		// 	if importStorage == importStorageStarted {
		// 		if strings.Contains(s, "# Fill in the empty properties") {
		// 			continue
		// 		}
		// 		if strings.HasPrefix(s, "#") {
		// 			if success, rErr := storage.Remap(g.StorageDomains); rErr != nil {
		// 				if success {
		// 					remapWarnings = append(remapWarnings, rErr)
		// 				} else {
		// 					err = rErr
		// 					return
		// 				}
		// 			}
		// 			if err = storage.Write(writer); err != nil {
		// 				return
		// 			}
		// 			importStorage = importStorageNone
		// 			if err = writer.WriteByte('\n'); err != nil {
		// 				return
		// 			}
		// 			if _, err = writer.WriteString(s); err != nil {
		// 				return
		// 			}
		// 			if err = writer.WriteByte('\n'); err != nil {
		// 				return
		// 			}
		// 			continue
		// 		}
		// 	}
		// 	if _, err = writer.WriteString(s); err != nil {
		// 		return
		// 	}
		// 	if err = writer.WriteByte('\n'); err != nil {
		// 		return
		// 	}
		// 	continue
		// } else if importStorage == importStorageWant {
		// 	if strings.HasPrefix(s, "- ") {
		// 		importStorage = importStorageStarted
		// 		storage.Set(utils.Clone(s[2:]))
		// 		continue
		// 	} else {
		// 		importStorage = importStorageNone
		// 	}
		// } else if importStorage == importStorageStarted {
		// 	if strings.HasPrefix(s, "  ") {
		// 		storage.Set(utils.Clone(s[2:]))
		// 		continue
		// 	} else if strings.HasPrefix(s, "- ") {
		// 		storage.Reset()
		// 		storage.Set(utils.Clone(s[2:]))
		// 		continue
		// 	} else if s == "" {
		// 		continue
		// 	} else {
		// 		if success, rErr := storage.Remap(g.StorageDomains); rErr != nil {
		// 			if success {
		// 				remapWarnings = append(remapWarnings, rErr)
		// 			} else {
		// 				err = rErr
		// 				return
		// 			}
		// 		}
		// 		if err = storage.Write(writer); err != nil {
		// 			return
		// 		}
		// 		importStorage = importStorageNone
		// 		if err = writer.WriteByte('\n'); err != nil {
		// 			return
		// 		}
		// 	}
		// }

		switch importStorage {
		case importStorageStarted:
			if strings.HasPrefix(s, "- ") {
				importStorage = importStorageWant
				indent = startBytes(s[1:], ' ') + 1
				if indent != 2 {
					err = ErrImportStorageItem
					return
				}
				storage.Set(s[indent:])
			} else {
				err = ErrImportStorageItem
				return
			}
		case importStorageWant:
			if strings.HasPrefix(s, "- ") {
				if storage.StorageType != "" {
					// flush map
					if success, rErr := storage.Remap(g.StorageDomains); rErr != nil {
						if success {
							remapWarnings = append(remapWarnings, rErr)
						} else {
							err = rErr
							return
						}
					}
					if err = storage.Write(writer); err != nil {
						return
					}
				}
				storage.Reset()

				importStorage = importStorageWant
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
				if storage.StorageType != "" {
					if success, rErr := storage.Remap(g.StorageDomains); rErr != nil {
						if success {
							remapWarnings = append(remapWarnings, rErr)
						} else {
							err = rErr
							return
						}
					}
					if err = storage.Write(writer); err != nil {
						return
					}
				}

				importStorage = importStorageNone
				storage.Reset()

				k, v, ok := splitKV(s, true)
				if ok {
					if err = writeKVLn(writer, k, v); err != nil {
						return
					}
				} else if err = writeStringLn(writer, s); err != nil {
					return
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

				importStorage = importStorageStarted
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

	if (importStorage == importStorageStarted || importStorage == importStorageWant) && storage.StorageType != "" {
		if success, rErr := storage.Remap(g.StorageDomains); rErr != nil {
			if success {
				remapWarnings = append(remapWarnings, rErr)
			} else {
				err = rErr
				return
			}
		}
		if err = storage.Write(writer); err != nil {
			return
		}
	}

	if err = writer.Flush(); err != nil {
		return
	}

	err = scanner.Err()
	if err != nil {
		return
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
