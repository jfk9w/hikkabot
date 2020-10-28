package reddit

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type Uint64Set map[uint64]bool

func (s Uint64Set) Has(key uint64) bool {
	return s[key]
}

func (s Uint64Set) Add(key uint64) {
	s[key] = true
}

type BigUint64Set struct {
	Base string        `json:"b,omitempty"`
	Off  flu.StringSet `json:"o,omitempty"`
}

func (s Uint64Set) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteRune('[')
	first := true
	for val := range s {
		if !first {
			b.WriteRune(',')
		} else {
			first = false
		}

		b.WriteString(strconv.FormatUint(val, 10))
	}

	b.WriteRune(']')

	return []byte(b.String()), nil
}

func (s Uint64Set) UnmarshalJSONv3(data []byte) error {
	str := string(data)
	str = str[1 : len(str)-1]
	if str == "" {
		return nil
	}

	start := 0
	lastIdx := len(str) - 1
	for i, c := range str {
		if c == ',' || i == lastIdx {
			if i == lastIdx {
				i += 1
			}

			token := strings.Trim(str[start:i], " ")
			val, err := strconv.ParseUint(token, 10, 64)
			if err != nil {
				return errors.Wrapf(err, "decode %s", token)
			}

			s.Add(val)
			start = i + 1
		}
	}

	return nil
}

func (s Uint64Set) UnmarshalJSONv2(data []byte) error {
	str := string(data[1 : len(data)-1])
	if str == "" {
		return nil
	}

	var base uint64 = math.MaxUint64
	start := 0
	lastIdx := len(str) - 1
	for i, c := range str {
		if c == ':' || i == lastIdx {
			if i == lastIdx {
				i += 1
			}

			token := str[start:i]
			val, err := DecodeString(token)
			if err != nil {
				return errors.Wrapf(err, "decode %s", token)
			}

			if base == math.MaxUint64 {
				base = val
				s.Add(base)
			} else {
				s.Add(base + val)
			}

			start = i + 1
		}
	}

	return nil
}

func (s Uint64Set) UnmarshalJSONv1(data []byte) error {
	repr := BigUint64Set{Off: make(flu.StringSet)}
	if err := json.Unmarshal(data, &repr); err != nil {
		return errors.Wrap(err, "unmarshal repr")
	}

	if string(data) == "{}" {
		return nil
	}

	base, err := strconv.ParseUint(repr.Base, 36, 64)
	if err != nil {
		return errors.Wrap(err, "parse avg")
	}

	s.Add(base)
	for str := range repr.Off {
		off, err := strconv.ParseUint(str, 36, 64)
		if err != nil {
			return errors.Wrapf(err, "parse offset: %s", str)
		}

		s.Add(base + off)
	}

	return nil
}

func (s Uint64Set) UnmarshalJSON(data []byte) error {
	if err := s.UnmarshalJSONv3(data); err != nil {
		if err := s.UnmarshalJSONv2(data); err != nil {
			return s.UnmarshalJSONv1(data)
		}
	}

	return nil
}

func (s Uint64Set) Copy() Uint64Set {
	copy := make(Uint64Set, len(s))
	for value := range s {
		copy.Add(value)
	}

	return copy
}

func EncodeToString(value uint64) string {
	return strconv.FormatUint(value, 36)
}

func DecodeString(str string) (uint64, error) {
	return strconv.ParseUint(str, 36, 64)
}
