package util

import (
	"io"

	yaml "gopkg.in/yaml.v2"
)

type YAML struct {
	Value interface{}
}

func (y YAML) ContentType() string {
	return "application/yaml"
}

func (y YAML) EncodeTo(w io.Writer) error {
	return yaml.NewEncoder(w).Encode(y.Value)
}

func (y YAML) DecodeFrom(r io.Reader) error {
	return yaml.NewDecoder(r).Decode(y.Value)
}
