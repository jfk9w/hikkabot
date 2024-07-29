package apfel

import (
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"io"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel/internal"
	"github.com/moul/flexyaml"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func init() {
	gob.Register(map[string]any{})
	gob.Register(map[any]any{})
	gob.Register([]any{})
	gob.Register(internal.AnyMap{})
}

// GobViaYAML is encoding/gob "frontend" for ViaYAML.
func GobViaYAML(value any) flu.ValueCodec {
	return viaYAML{
		value:   value,
		encoder: func(w io.Writer) Encoder { return gob.NewEncoder(w) },
		decoder: func(r io.Reader) Decoder { return gob.NewDecoder(r) },
	}
}

// JSONViaYAML is encoding/json "frontend" for ViaYAML.
func JSONViaYAML(value any) flu.ValueCodec {
	return viaYAML{
		value:   value,
		encoder: func(w io.Writer) Encoder { return json.NewEncoder(w) },
		decoder: func(r io.Reader) Decoder { return json.NewDecoder(r) },
	}
}

// XMLViaYAML is encoding/xml "frontend" for ViaYAML.
func XMLViaYAML(value any) flu.ValueCodec {
	return viaYAML{
		value:   value,
		encoder: func(w io.Writer) Encoder { return xml.NewEncoder(w) },
		decoder: func(r io.Reader) Decoder { return xml.NewDecoder(r) },
	}
}

type (
	// Encoder may be encoded a value to.
	Encoder interface{ Encode(any) error }
	// EncoderFunc in the Encoder functional adapter.
	EncoderFunc func() Encoder
	// Decoder may decoded into a value.
	Decoder interface{ Decode(any) error }
	// DecoderFunc is the Decoder functional adapter.
	DecoderFunc func() Decoder
)

type viaYAML struct {
	value   any
	encoder func(io.Writer) Encoder
	decoder func(io.Reader) Decoder
}

// ViaYAML returns a flu.Codec which performs marshalling and unmarshalling via YAML codec.
// Marshal steps: marshal value with YAML, decode as YAML into internal.AnyMap and encode the map with encoder function.
// Unmarshal steps: unmarshal internal.AnyMap with decoder function, marshal it with YAML and unmarshal value as YAML.
// This provides the ability to use a single tag (yaml) for field declarations for all codecs.
func ViaYAML(encoder func(io.Writer) Encoder, decoder func(io.Reader) Decoder) flu.Codec {
	return func(value any) flu.ValueCodec {
		return viaYAML{
			value:   value,
			encoder: encoder,
			decoder: decoder,
		}
	}
}

func (v viaYAML) EncodeTo(writer io.Writer) error {
	data, err := yaml.Marshal(v.value)
	if err != nil {
		return errors.Wrap(err, "marshal yaml")
	}

	values := make(internal.AnyMap)
	if err := flexyaml.Unmarshal(data, &values); err != nil {
		return errors.Wrap(err, "unmarshal yaml values")
	}

	values.SpecifyTypes()
	encoder := v.encoder(writer)
	defer flu.CloseQuietly(encoder)
	return encoder.Encode(values)
}

func (v viaYAML) DecodeFrom(reader io.Reader) error {
	decoder := v.decoder(reader)
	defer flu.CloseQuietly(decoder)

	values := make(internal.AnyMap)
	if err := decoder.Decode(&values); err != nil {
		return errors.Wrap(err, "unmarshal json values")
	}

	values.SpecifyTypes()
	data, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "marshal yaml values")
	}

	return yaml.Unmarshal(data, v.value)
}
