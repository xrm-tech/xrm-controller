package ovirt

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/containers/storage/pkg/lockfile"
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

// GenerateVars is OVirt engines API address/credentials
type GenerateVars struct {
	PrimaryUrl        string `json:"site_primary_url" validate:"required,startswith=https://"`
	PrimaryUsername   string `json:"site_primary_username" validate:"required"`
	PrimaryPassword   string `json:"site_primary_password" validate:"required"`
	SecondaryUrl      string `json:"site_secondary_url" validate:"required,startswith=https://"`
	SecondaryUsername string `json:"site_secondary_username" validate:"required"`
	SecondaryPassword string `json:"site_secondary_password" validate:"required"`
}

func (g GenerateVars) Generate(name, dir string) (out string, err error) {
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

	var lock *lockfile.LockFile
	if lock, err = lockfile.GetLockFile(dir + ".lock"); err != nil {
		return
	}
	lock.Lock()
	defer lock.Unlock()

	ansibleGeneratePlaybook := path.Join(dir, ansibleGeneratePlaybook)
	ansibleFailoverPlaybook := path.Join(dir, ansibleFailoverPlaybook)
	ansibleFailbackPlaybook := path.Join(dir, ansibleFailbackPlaybook)
	ansibleVarFile := path.Join(dir, ansibleDrVarsFile)
	ansibleVarFileTpl := ansibleVarFile + ".tpl"
	primaryCaFile := path.Join(dir, "primary.ca")
	secondaryCaFile := path.Join(dir, "secondary.ca")

	if err = cp.Copy(template, dir); err != nil {
		return
	}

	if err = saveCaFile(g.PrimaryUrl, primaryCaFile); err != nil {
		return
	}
	if err = saveCaFile(g.SecondaryUrl, secondaryCaFile); err != nil {
		return
	}

	if err = validateOvirtCon(g.PrimaryUrl, false, primaryCaFile, g.PrimaryUsername, g.PrimaryPassword); err != nil {
		return
	}

	if err = g.writeAnsiblePwdDile(path.Join(dir, ansibleDrPwdFile)); err != nil {
		return
	}

	extraVars := "site=" + g.PrimaryUrl + " username=" + g.PrimaryUsername + " password=" + g.PrimaryPassword +
		" ca=" + primaryCaFile + " var_file=" + ansibleVarFileTpl

	// TODO: reduce verbose ?
	if out, err = utils.ExecCmd(time.Minute*2, "ansible-playbook", ansibleGeneratePlaybook, "-t", ansibleDrTag, "-e", extraVars, "-vvvvv"); err == nil {
		if utils.FileExists(ansibleVarFileTpl) {
			err = g.writeAnsibleVarsFile(ansibleVarFileTpl, ansibleVarFile)
		} else {
			err = ErrVarFileNotExist
		}
	}

	if err == nil {
		err = g.writeAnsibleFailbackFile(ansibleFailoverPlaybook, ansibleFailbackPlaybook)
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

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		s := scanner.Text()
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
				if _, err = writer.WriteString(v); err != nil {
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
