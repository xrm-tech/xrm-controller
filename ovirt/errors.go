package ovirt

import (
	"errors"
)

var (
	ErrAnsibleFileNoStorages = errors.New("ansible file has no storages")
	ErrNameInvalid           = errors.New("name is invalid")
	ErrDirNotExist           = errors.New("dir not exist, run generate")
	ErrInProgress            = errors.New("another operation in progress")
	ErrAnsibleNotFound       = errors.New("ansible-playbook not found")
)
