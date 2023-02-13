package ovirt

import (
	"os"

	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

// Cleanup delete ovirt/{name}/disaster_recovery_vars.yml
func Cleanup(path string) (err error) {
	if utils.FileExists(path) {
		err = os.Remove(path)
	}
	return
}
