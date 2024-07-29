package me3x

import (
	"fmt"
	"strings"
)

// Labeled is a value which has Labels.
type Labeled interface {
	Labels() Labels
}

// Label is a key-value pair.
type Label struct {

	// Name is the Label name.
	Name string

	// Value is the Label value.
	Value any
}

// Labels is the Label slice.
type Labels []Label

// Name adds a Label with an empty Value to Labels.
func (l Labels) Name(name string) Labels {
	return append(l, Label{Name: name})
}

// Add adds a Label to Labels.
func (l Labels) Add(name string, value any) Labels {
	return append(l, Label{Name: name, Value: value})
}

// AddAll adds all Labels from the other slice to this one.
func (l Labels) AddAll(labels Labels) Labels {
	return append(l, labels...)
}

// Names returns all Label names.
func (l Labels) Names() []string {
	if l == nil {
		return nil
	}
	keys := make([]string, len(l))
	for i, label := range l {
		keys[i] = label.Name
	}

	return keys
}

// StringMap returns a string-string map based on the labels.
func (l Labels) StringMap() map[string]string {
	result := make(map[string]string, len(l))
	for _, label := range l {
		result[label.Name] = fmt.Sprint(label.Value)
	}

	return result
}

// Map returns a string-any map based on the labels.
func (l Labels) Map() map[string]any {
	result := make(map[string]any, len(l))
	for _, label := range l {
		result[label.Name] = label.Value
	}

	return result
}

func (l Labels) String() string {
	var b strings.Builder
	b.WriteRune('{')
	for i, l := range l {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(l.Name)
		b.WriteRune('=')
		b.WriteString(fmt.Sprintf("%v", l.Value))
	}

	b.WriteRune('}')
	return b.String()
}

func (l Labels) graphitePath(sep, esc string) string {
	b := new(strings.Builder)
	first := true
	for _, label := range l {
		var value string
		if label.Value != nil {
			value = fmt.Sprint(label.Value)
		} else {
			value = label.Name
		}

		if first {
			first = false
		} else {
			b.WriteRune('.')
		}

		value = strings.Replace(value, sep, esc, -1)
		b.WriteString(value)
	}

	return b.String()
}

func withPrefix(base, suffix, sep string) string {
	if base != "" {
		base += sep
	}

	return base + suffix
}
