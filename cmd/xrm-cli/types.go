package main

import (
	"fmt"
)

type xrmType uint8

const (
	xrmOVirt xrmType = iota
)

func newXrmType(val xrmType, p *xrmType) *xrmType {
	*p = val
	return p
}

var xrmTypeStrings []string = []string{"ovirt"}

func (t *xrmType) Set(value string, _ bool) error {
	switch value {
	case "ovirt":
		*t = xrmOVirt
	default:
		return fmt.Errorf("invalid type %s", value)
	}
	return nil
}

func (t *xrmType) String() string {
	return xrmTypeStrings[*t]
}

func (*xrmType) Type() string {
	return "type"
}

func (t *xrmType) Reset(i interface{}) {
	v := i.(xrmType)
	*t = xrmType(v)
}

func (t *xrmType) Get() interface{} {
	return *t
}
