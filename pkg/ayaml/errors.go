package ayaml

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrorUnclosedYAML = errors.New("yaml file not ended as expected")

type ErrorParse struct {
	v    string
	msg  string
	line uint
}

func NewParseError(v, msg string, line uint) ErrorParse {
	return ErrorParse{v: v, msg: msg, line: line}
}

func (e ErrorParse) Error() string {
	return e.msg + " (line " + strconv.FormatUint(uint64(e.line), 10) + "): '" + e.v + "'"
}

type ErrorIndent struct {
	v      string
	msg    string
	indent int
	line   uint
}

func NewIndentError(v, msg string, indent int, line uint) ErrorIndent {
	return ErrorIndent{v: v, msg: msg, indent: indent, line: line}
}

func (e ErrorIndent) Error() string {
	return fmt.Sprintf("%s, indent %d is invalid (line %d): '%s'", e.msg, e.indent, e.line, e.v)
}

type ErrorParsePanic struct {
	msg        string
	stackTrace string
	line       uint
}

func NewParsePanicError(msg string, line uint, stackTrace string) ErrorParsePanic {
	return ErrorParsePanic{msg: msg, line: line, stackTrace: stackTrace}
}

func (e ErrorParsePanic) Error() string {
	return fmt.Sprintf("%s (line %d): %+v", e.msg, e.line, e.stackTrace)
}

func RecoverToError(r interface{}, line uint, stackTrace string) error {
	switch x := r.(type) {
	case string:
		return NewParsePanicError(x, line, stackTrace)
	case error:
		return NewParsePanicError(x.Error(), line, stackTrace)
	default:
		return NewParsePanicError(fmt.Sprint(x), line, stackTrace)
	}
}
