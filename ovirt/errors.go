package ovirt

import "errors"

var (
	ErrNameInvalid = errors.New("name is invalid")
	ErrDirNotExist = errors.New("dir not exist, run generate")
	ErrInProgress  = errors.New("another operation in progress")
)
