package xrmcontroller

import (
	"bytes"

	"github.com/goccy/go-json"
)

func Decode(data []byte, v interface{}) error {
	r := bytes.NewBuffer(data)
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
