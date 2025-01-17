package executor

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
	"gopkg.in/yaml.v2"
)

var (
	ErrNotSupported = errors.New("method not supported")
	ErrPathNotFound = errors.New("path not found")
)

// Delete delete {dir}/{name}
func Delete(name, dir string) (err error) {
	path := path.Join(dir, name) + ".plandata"

	return os.Remove(path)
}

// GenerateVars is OVirt engines API address/credentials
type GenerateVars struct {
	PrimaryIP              string `json:"01_broker_primary_ip" yaml:"01_broker_primary_ip"`
	PrimaryUsername        string `json:"02_broker_primary_username" yaml:"02_broker_primary_username"`
	PrimaryAuthenticator   string `json:"03_broker_primary_authenticator" yaml:"03_broker_primary_authenticator"`
	PrimaryPassword        string `json:"04_broker_primary_password" yaml:"04_broker_primary_password"`
	SecondaryIP            string `json:"05_broker_secondary_ip" yaml:"05_broker_secondary_ip"`
	SecondaryUsername      string `json:"06_broker_secondary_username" yaml:"06_broker_secondary_username"`
	SecondaryPassword      string `json:"08_broker_secondary_password" yaml:"08_broker_secondary_password"`
	SecondaryAuthenticator string `json:"07_broker_secondary_authenticator" yaml:"07_broker_secondary_authenticator"`

	ServicePoolName string `json:"09_service_pool_name" yaml:"09_service_pool_name"`
}

func (g GenerateVars) Validate() error {
	var errs utils.Errors

	if g.PrimaryIP == "" {
		errs = append(errs, "01_broker_primary_ip is empty")
	}
	if g.PrimaryUsername == "" {
		errs = append(errs, "02_broker_primary_username is empty")
	}
	if g.PrimaryPassword == "" {
		errs = append(errs, "site_primary_password is empty")
	}

	if g.SecondaryIP == "" {
		errs = append(errs, "05_broker_secondary_ip is empty")
	}
	if g.SecondaryUsername == "" {
		errs = append(errs, "06_broker_secondary_username is empty")
	}
	if g.SecondaryPassword == "" {
		errs = append(errs, "08_broker_secondary_password is empty")
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func Generate(name, dir string, cfg GenerateVars) (out string, err error) {
	path, err := exec.LookPath("xrm-openuds-generate.py")
	if err != nil {
		err = ErrPathNotFound
		return
	}

	in, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", err
	}

	return utils.ExecCmdIn(bytes.NewReader(in), dir+"/generate.log", time.Minute*10, path, "-n", name)
}

func Failover(name, dir string) (out string, err error) {
	path, err := exec.LookPath("xrm-openuds-failover.py")
	if err != nil {
		err = ErrPathNotFound
		return
	}

	return utils.ExecCmd(dir+"/generate.log", time.Minute*10, path, "-n", name)
}
