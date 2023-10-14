package ayaml

import (
	"regexp"
	"strings"
)

var (
	commentRe = regexp.MustCompile(`^ *#`)
)

// type valType uint8

// const (
// 	valString valType = iota
// 	valValue
// 	valListStart
// )

func split(s string) (k string, v string, ok, commented bool) {
	if commentRe.MatchString(s) {
		return
	}
	k, v, ok = strings.Cut(s, ":")
	if ok {
		k = strings.TrimRight(k, " ")
		v = strings.TrimLeft(v, " ")
		if strings.HasPrefix(v, "#") {
			commented = true
			v = strings.TrimLeft(v, "#")
			v = strings.TrimLeft(v, " ")
		}
	}
	return
}

func splitIndent(s string, line uint) (v string, indent int, listStart bool, err error) {
	v = strings.TrimLeft(s, " ")
	if strings.HasPrefix(v, "- ") {
		listStart = true
		v = v[2:]
	}
	if strings.HasPrefix(v, " ") {
		err = NewParseError(s, "more than one space after -", line)
	}
	indent = len(s) - len(v)

	return
}
