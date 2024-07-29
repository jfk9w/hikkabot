package colf

import (
	"encoding/json"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Set contains non-repeating values.
type Set[E comparable] map[E]bool

func (s Set[E]) MarshalJSON() ([]byte, error) {
	return json.Marshal(ToSlice[E](s))
}

func (s *Set[E]) UnmarshalJSON(data []byte) error {
	var slice Slice[E]
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	AddAll[E](s, slice)
	return nil
}

func (s Set[E]) MarshalYAML() (interface{}, error) {
	return ToSlice[E](s), nil
}

func (s *Set[E]) UnmarshalYAML(node *yaml.Node) error {
	var slice []E
	if data, err := yaml.Marshal(node); err != nil {
		return errors.Wrap(err, "marshal yaml node")
	} else if err := yaml.Unmarshal(data, &slice); err != nil {
		return err
	}

	AddAll[E](s, Slice[E](slice))
	return nil
}

func (s *Set[E]) Add(element E) {
	if *s == nil {
		*s = make(Set[E])
	}

	(*s)[element] = true
}

func (s Set[E]) ForEach(forEach ForEach[E]) {
	for element := range s {
		if !forEach(element) {
			break
		}
	}
}

func (s Set[E]) Size() int {
	return len(s)
}
