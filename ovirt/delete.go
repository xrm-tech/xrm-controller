package ovirt

import (
	"os"
	"path"

	"github.com/juju/fslock"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

// Delete delete {dir}/{name}
func Delete(name, dir string) (err error) {
	if !validateName(name) {
		return ErrNameInvalid
	}

	dir = path.Join(dir, name)

	if utils.DirExists(dir) {
		if !lock.TryLock() {
			return ErrInProgress
		}
		defer lock.Unlock()

		flock := fslock.New(dir + ".lock")
		if err = flock.TryLock(); err != nil {
			return
		}
		defer func() { _ = flock.Unlock() }()

		err = os.RemoveAll(dir)
	}
	return
}
