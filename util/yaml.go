package util

import (
	"io"

	"github.com/jfk9w-go/flu"

	yaml "gopkg.in/yaml.v2"
)

type yamlBody struct {
	value interface{}
}

func YAML(value interface{}) flu.BodyReadWriter {
	return yamlBody{value}
}

func (b yamlBody) ContentType() string {
	return "application/yaml"
}

func (b yamlBody) WriteTo(w io.Writer) error {
	return yaml.NewEncoder(w).Encode(b.value)
}

func (b yamlBody) ReadFrom(r io.Reader) error {
	return yaml.NewDecoder(r).Decode(b.value)
}
