package utils

import "strings"

type Errors []string

func (errs Errors) Error() string {
	var buf strings.Builder
	for _, e := range errs {
		buf.WriteString(e)
		buf.WriteByte('\n')
	}
	return buf.String()
}
