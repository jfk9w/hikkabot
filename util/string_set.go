package util

import (
	"encoding/json"
)

type StringSet map[string]bool

func (s StringSet) Has(key string) bool {
	return s[key]
}

func (s StringSet) Add(key string) {
	s[key] = true
}

func (s StringSet) Delete(key string) {
	delete(s, key)
}

func (s StringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Slice())
}

func (s StringSet) UnmarshalJSON(data []byte) error {
	slice := make([]string, 0)
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	s.Fill(slice)
	return nil
}

func (s StringSet) Copy() StringSet {
	copy := make(StringSet, len(s))
	for value := range s {
		copy.Add(value)
	}

	return copy
}

func (s StringSet) Slice() []string {
	if len(s) == 0 {
		return nil
	}

	slice := make([]string, len(s))
	i := 0
	for value := range s {
		slice[i] = value
		i++
	}

	return slice
}

func (s StringSet) Fill(slice []string) {
	for _, value := range slice {
		s.Add(value)
	}
}
