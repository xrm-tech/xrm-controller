package ovirt

import (
	"os"
	"path"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

// Delete delete {dir}/{name}
func Delete(name, dir string) (err error) {
	if !validateName(name) {
		return ErrNameInvalid
	}

	dir = path.Join(dir, name)

	if utils.DirExists(dir) {
		err = os.RemoveAll(dir)
	}
	return
}
