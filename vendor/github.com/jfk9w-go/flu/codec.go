package flu

import (
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"io"

	"gopkg.in/yaml.v3"
)

// JSON encodes/decodes the provided ValueCodec using JSON format.
func JSON(value interface{}) ValueCodec {
	return jsonValue{value}
}

type jsonValue struct {
	value interface{}
}

func (v jsonValue) EncodeTo(w io.Writer) error {
	return json.NewEncoder(w).Encode(v.value)
}

func (v jsonValue) DecodeFrom(r io.Reader) error {
	return json.NewDecoder(r).Decode(v.value)
}

func (v jsonValue) ContentType() string {
	return "application/json"
}

func XML(value interface{}) ValueCodec {
	return xmlValue{value}
}

// XML encodes/decodes the provided value using XML format.
type xmlValue struct {
	value interface{}
}

func (v xmlValue) EncodeTo(w io.Writer) error {
	return xml.NewEncoder(w).Encode(v.value)
}

func (v xmlValue) DecodeFrom(r io.Reader) error {
	return xml.NewDecoder(r).Decode(v.value)
}

func (v xmlValue) ContentType() string {
	return "application/xml"
}

// Text encodes/decodes the provided value as plain text.
type Text struct {
	Value string
}

func (v Text) EncodeTo(w io.Writer) error {
	_, err := io.WriteString(w, v.Value)
	return err
}

func (v *Text) DecodeFrom(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	v.Value = string(data)
	return nil
}

func (v *Text) ContentType() string {
	return "text/plain; charset=utf-8"
}

func YAML(value interface{}) ValueCodec {
	return yamlValue{value}
}

// YAML encodes/decodes the provided value using YAML format.
type yamlValue struct {
	value interface{}
}

func (v yamlValue) EncodeTo(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer CloseQuietly(enc)
	enc.SetIndent(2)
	return enc.Encode(v.value)
}

func (v yamlValue) DecodeFrom(r io.Reader) error {
	return yaml.NewDecoder(r).Decode(v.value)
}

func Gob(value interface{}) ValueCodec {
	return gobValue{value}
}

type gobValue struct {
	value interface{}
}

func (v gobValue) EncodeTo(writer io.Writer) error {
	return gob.NewEncoder(writer).Encode(v.value)
}

func (v gobValue) DecodeFrom(reader io.Reader) error {
	return gob.NewDecoder(reader).Decode(v.value)
}
