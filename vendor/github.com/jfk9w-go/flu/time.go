package flu

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

// TimeLayout is the default layout used while parsing Time from a string.
var TimeLayout = "2006-01-02 15:04:05"

// Time is time.Time wrapper with serialization support.
type Time struct {
	time.Time

	// Layout may be used to custom string value layout.
	Layout string
}

func (t Time) String() string {
	return t.Time.Format(TimeLayout)
}

// FromString parses a string.
func (t *Time) FromString(str string) error {
	layout := t.Layout
	if layout == "" {
		layout = TimeLayout
	}

	value, err := time.Parse(layout, str)
	if err != nil {
		return errors.Wrap(err, "parse time")
	}
	t.Time = value
	return nil
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return errors.Wrap(err, "unmarshal")
	}
	return t.FromString(str)
}

func (t *Time) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

func (t *Time) UnmarshalYAML(node *yaml.Node) error {
	return t.FromString(node.Value)
}
