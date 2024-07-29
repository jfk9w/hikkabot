package flu

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

// Duration provides useful serialization methods for time.Duration.
type Duration struct {
	Value time.Duration
}

// FromString parses the duration from a string value.
func (d *Duration) FromString(str string) error {
	value, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	if d == nil {
		*d = Duration{}
	}

	(*d).Value = value
	return err
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	return d.FromString(node.Value)
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Value.String(), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return errors.Wrap(err, "unmarshal string")
	}

	return d.FromString(str)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Value.String())
}
