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

	"github.com/juju/fslock"
	cp "github.com/otiai10/copy"
	ovirtsdk4 "github.com/ovirt/go-ovirt"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	ErrTemplateDirNotExist = errors.New("ovirt template dir not exist")
	ErrDirAlreadyExist     = errors.New("dir already exist")
	ErrVarFileNotExist     = errors.New("var file not exist")
	// ansible
	ansibleDrTag            = "generate_mapping"
	ansibleGeneratePlaybook = "dr_generate.yml"
	ansibleFailoverPlaybook = "dr_failover.yml"
	ansibleFailbackPlaybook = "dr_failback.yml"
	ansibleDrVarsFileTpl    = "disaster_recovery_vars.yml.tpl"
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

type Storage struct {
	StorageType   string   `json:"storage_type" validate:"required"`
	PrimaryName   string   `json:"primary_name" validate:"required"`
	PrimaryDC     string   `json:"-"`
	PrimaryPath   string   `json:"primary_path" validate:"required"`
	PrimaryAddr   string   `json:"primary_addr" validate:"required"`
	SecondaryName string   `json:"secondary_name"`
	SecondaryDC   string   `json:"secondary_dc_name"`
	SecondaryPath string   `json:"secondary_path" validate:"required"`
	SecondaryAddr string   `json:"secondary_addr" validate:"required"`
	Additional    []string `json:"-"`
}

func (m *Storage) Reset() {
	m.StorageType = ""
	m.PrimaryName = ""
	m.PrimaryPath = ""
	m.PrimaryAddr = ""
	m.SecondaryName = ""
	m.SecondaryPath = ""
	m.SecondaryAddr = ""
	m.Additional = m.Additional[:0]
}

func (m *Storage) Set(s string) {
	k, v, ok := strings.Cut(s, ": ")
	if !ok {
		m.Additional = append(m.Additional, s)
	}
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
}

func (m *Storage) Remap(storageDomains []Storage) error {
	for _, domain := range storageDomains {
		if m.StorageType == domain.StorageType && m.PrimaryName == domain.PrimaryName &&
			m.PrimaryAddr == domain.PrimaryAddr && m.PrimaryPath == domain.PrimaryPath {
			if domain.SecondaryName == "" {
				m.SecondaryName = domain.PrimaryName
			} else {
				m.SecondaryName = domain.SecondaryName
			}
			if domain.SecondaryDC == "" {
				m.SecondaryDC = m.PrimaryDC
			} else {
				m.SecondaryDC = domain.SecondaryDC
			}
			m.SecondaryAddr = domain.SecondaryAddr
			m.SecondaryPath = domain.SecondaryPath
			return nil
		}
	}
	return errors.New("storage map for " + m.PrimaryName + " not found")
}

func writeEntry(w *bufio.Writer, k, v string) error {
	if _, err := w.WriteString(k); err != nil {
		return err
	}
	if _, err := w.WriteString(v); err != nil {
		return err
	}
	return w.WriteByte('\n')
}

func (m *Storage) Write(w *bufio.Writer) error {
	if err := writeEntry(w, "- dr_domain_type: ", m.StorageType); err != nil {
		return err
	}

	if err := writeEntry(w, "  dr_primary_name: ", m.PrimaryName); err != nil {
		return err
	}
	if err := writeEntry(w, "  dr_primary_dc_name: ", m.PrimaryDC); err != nil {
		return err
	}
	if err := writeEntry(w, "  dr_primary_path: ", m.PrimaryPath); err != nil {
		return err
	}
	if err := writeEntry(w, "  dr_primary_address: ", m.PrimaryAddr); err != nil {
		return err
	}

	if err := writeEntry(w, "  dr_secondary_name: ", m.SecondaryName); err != nil {
		return err
	}
	if err := writeEntry(w, "  dr_secondary_dc_name: ", m.SecondaryDC); err != nil {
		return err
	}
	if err := writeEntry(w, "  dr_secondary_path: ", m.SecondaryPath); err != nil {
		return err
	}
	if err := writeEntry(w, "  dr_secondary_address: ", m.SecondaryAddr); err != nil {
		return err
	}

	for _, a := range m.Additional {
		if err := writeEntry(w, "  ", a); err != nil {
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
			flock.Unlock()
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
					err = g.writeAnsibleVarsFile(ansibleVarFileTpl, ansibleVarFile)
				}
			} else {
				err = ErrVarFileNotExist
			}
		}
	}()

	wg.Wait()

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

var blankRe = regexp.MustCompile(`^ *#`)

func (g GenerateVars) writeAnsibleVarsFile(template, varFile string) (err error) {
	var in, out *os.File
	in, err = os.Open(template)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err = os.OpenFile(varFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := bufio.NewWriter(out)

	var (
		importStorage importStorageState
		storage       Storage
		remapErrs     []error
	)
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		s := scanner.Text()
		if blankRe.MatchString(s) {
			if importStorage == importStorageStarted {
				if strings.Contains(s, "# Fill in the empty properties") {
					continue
				}
				if strings.HasPrefix(s, "#") {
					if rErr := storage.Remap(g.StorageDomains); rErr != nil {
						remapErrs = append(remapErrs, rErr)
					}
					if err = storage.Write(writer); err != nil {
						return err
					}
					importStorage = importStorageNone
					if err = writer.WriteByte('\n'); err != nil {
						return
					}
					if _, err = writer.WriteString(s); err != nil {
						return
					}
					if err = writer.WriteByte('\n'); err != nil {
						return
					}
					continue
				}
			}
			if _, err = writer.WriteString(s); err != nil {
				return
			}
			if err = writer.WriteByte('\n'); err != nil {
				return
			}
			continue
		} else if importStorage == importStorageWant {
			if strings.HasPrefix(s, "- ") {
				importStorage = importStorageStarted
				storage.Set(utils.Clone(s[2:]))
				continue
			} else {
				importStorage = importStorageNone
			}
		} else if importStorage == importStorageStarted {
			if strings.HasPrefix(s, "  ") {
				storage.Set(utils.Clone(s[2:]))
				continue
			} else if strings.HasPrefix(s, "- ") {
				storage.Reset()
				storage.Set(utils.Clone(s[2:]))
				continue
			} else if s == "" {
				continue
			} else {
				if rErr := storage.Remap(g.StorageDomains); rErr != nil {
					remapErrs = append(remapErrs, rErr)
				}
				if err = storage.Write(writer); err != nil {
					return err
				}
				importStorage = importStorageNone
				if err = writer.WriteByte('\n'); err != nil {
					return
				}
			}
		}

		if strings.HasPrefix(s, "dr_sites_secondary_url: ") {
			if _, err = writer.WriteString("dr_sites_secondary_url: "); err != nil {
				return
			}
			if _, err = writer.WriteString(g.SecondaryUrl); err != nil {
				return
			}
			if err = writer.WriteByte('\n'); err != nil {
				return
			}
		} else if (strings.HasPrefix(s, "dr_sites_secondary_") || strings.HasPrefix(s, "  dr_secondary_") ||
			strings.HasPrefix(s, "  secondary_")) && strings.Contains(s, ": # ") {
			if k, v, ok := strings.Cut(s, ": # "); ok {
				if _, err = writer.WriteString(k); err != nil {
					return
				}
				if _, err = writer.WriteString(": "); err != nil {
					return
				}
				if strings.HasPrefix(s, "dr_sites_secondary_ca_file: ") {
					if _, err = writer.WriteString(strings.Replace(v, "primary.ca", "secondary.ca", 1)); err != nil {
						return
					}
				} else if _, err = writer.WriteString(v); err != nil {
					return
				}
				if err = writer.WriteByte('\n'); err != nil {
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
		} else if s == "dr_import_storages:" {
			if _, err = writer.WriteString(s); err != nil {
				return
			}
			if err = writer.WriteByte('\n'); err != nil {
				return
			}
			importStorage = importStorageWant
			storage.Reset()
		} else {
			if _, err = writer.WriteString(s); err != nil {
				return
			}
			if err = writer.WriteByte('\n'); err != nil {
				return
			}
		}
	}

	if importStorage == importStorageStarted {
		if rErr := storage.Remap(g.StorageDomains); rErr != nil {
			remapErrs = append(remapErrs, rErr)
		}
		if err = storage.Write(writer); err != nil {
			return err
		}
	}

	if err = writer.Flush(); err != nil {
		return
	}

	err = scanner.Err()
	if err != nil {
		return
	}
	if len(remapErrs) > 0 {
		err = remapErrs[0]
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
