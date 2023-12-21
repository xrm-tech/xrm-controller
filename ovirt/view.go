package ovirt

import (
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/juju/fslock"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

func ViewAll(dir string) (out string, err error) {
	if utils.DirExists(dir) {
		if !lock.TryLock() {
			return "", ErrInProgress
		}
		defer lock.Unlock()

		flock := fslock.New(dir + ".lock")
		if err = flock.TryLock(); err != nil {
			return
		}
		defer func() { _ = flock.Unlock() }()

		var (
			entries []fs.DirEntry
			b       strings.Builder
		)
		entries, err = os.ReadDir(dir)
		if err != nil {
			return
		}

		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				if name != "template" {
					if utils.FileExists(path.Join(dir, name, ansibleDrVarsFile)) {
						b.WriteString(name)
					} else {
						b.WriteString(name)
						b.WriteString(" (INCOMPLETE)")
					}
					b.WriteByte('\n')
				}
			}
		}
		out = b.String()
	}
	return
}

func View(name, dir string) (out string, err error) {
	if !validateName(name) {
		return "", ErrNameInvalid
	}

	if !lock.TryLock() {
		return "", ErrInProgress
	}
	defer lock.Unlock()

	flock := fslock.New(dir + ".lock")
	if err = flock.TryLock(); err != nil {
		return
	}
	defer func() { _ = flock.Unlock() }()

	path := path.Join(dir, name, ansibleDrVarsFile)
	if utils.FileExists(path) {
		var b []byte
		b, err = os.ReadFile(path)
		out = utils.UnsafeString(b)
	} else {
		err = ErrAnsibleDrVarsFile
	}

	return
}
