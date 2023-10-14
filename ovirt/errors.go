package ovirt

import (
	"errors"
	"strings"
)

var (
	ErrAnsibleFileNoStorages = errors.New("ansible file has no storages")
	ErrNameInvalid           = errors.New("name is invalid")
	ErrDirNotExist           = errors.New("dir not exist, run generate")
	ErrInProgress            = errors.New("another operation in progress")
	ErrAnsibleNotFound       = errors.New("ansible-playbook not found")
)

type Errors []string

func (errs Errors) Error() string {
	var buf strings.Builder
	for _, e := range errs {
		buf.WriteString(e)
		buf.WriteByte('\n')
	}
	return buf.String()
}
