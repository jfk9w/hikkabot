package reddit

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Uint64Set map[uint64]bool

func (s Uint64Set) Has(key uint64) bool {
	return s[key]
}

func (s Uint64Set) Add(key uint64) {
	s[key] = true
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

func (s Uint64Set) UnmarshalJSON(data []byte) error {
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
