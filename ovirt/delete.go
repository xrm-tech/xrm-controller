package ovirt

import (
	"os"
	"path"

	"github.com/containers/storage/pkg/lockfile"
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

		var flock *lockfile.LockFile
		if flock, err = lockfile.GetLockFile(dir + ".lock"); err != nil {
			return
		}
		flock.Lock()
		defer flock.Unlock()

		err = os.RemoveAll(dir)
	}
	return
}
