package ovirt

import (
	"path"
	"time"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	drCleanTag = "clean_engine"
)

// Failover initiate failover for {dir}/{name}
func Failover(name, dir string) (out string, err error) {
	if !validateName(name) {
		return "", ErrNameInvalid
	}

	dir = path.Join(dir, name)
	if !utils.DirExists(dir) {
		return "", ErrDirNotExist
	}

	playbook := path.Join(dir, ansibleFailoverPlaybook)

	// TODO: reduce verbose ?
	out, err = utils.ExecCmd(time.Minute*2, "ansible-playbook", playbook, "-t", drCleanTag, "-vvvvv")

	return string(out), err
}

// Failback initiate failback for {dir}/{name}
func Failback(name, dir string) (out string, err error) {
	if !validateName(name) {
		return "", ErrNameInvalid
	}

	dir = path.Join(dir, name)
	if !utils.DirExists(dir) {
		return "", ErrDirNotExist
	}

	playbook := path.Join(dir, ansibleFailbackPlaybook)

	// TODO: reduce verbose ?
	return utils.ExecCmd(time.Minute*2, "ansible-playbook", playbook, "-t", drCleanTag, "-vvvvv")
}
