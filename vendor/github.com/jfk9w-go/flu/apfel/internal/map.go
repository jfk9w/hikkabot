package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

func DecodeAsYAML(source, target any) error {
	return DecodeAs(source, target, flu.YAML)
}

func DecodeAs(source, target any, codec flu.Codec) error {
	var b bytes.Buffer
	if err := codec(source).EncodeTo(&b); err != nil {
		return errors.Wrap(err, "encode source")
	}

	return codec(target).DecodeFrom(&b)
}

var ErrKeyNotFound = errors.New("key not found")

type Map = map[string]any

type AnyMap map[any]any

func AnyMapFrom(from Map) AnyMap {
	if from == nil {
		return nil
	}

	r := make(AnyMap, len(from))
	for k, v := range from {
		r[k] = v
	}

	r.SpecifyTypes()
	return r
}

func (m AnyMap) As(v any, path ...any) error {
	if m == nil {
		if len(path) == 0 {
			return nil
		} else {
			return ErrKeyNotFound
		}
	}

	if len(path) == 0 {
		return DecodeAsYAML(m, v)
	}

	pathElement, _ := SpecifyType(path[0])

	if child, ok := m[pathElement]; ok {
		if len(path) == 1 {
			return DecodeAsYAML(child, v)
		}

		if child, ok := child.(AnyMap); ok {
			return errors.Wrapf(child.As(v, path[1:]...), ".%v", pathElement)
		}
	}

	return errors.Wrapf(ErrKeyNotFound, ".%v", pathElement)
}

func (m AnyMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Map())
}

func (m *AnyMap) UnmarshalJSON(data []byte) error {
	return m.unmarshal(data, json.Unmarshal)
}

func (m *AnyMap) unmarshal(data []byte, decode func([]byte, any) error) error {
	var from Map
	if err := decode(data, &from); err != nil {
		return err
	}

	if *m == nil {
		*m = make(AnyMap, len(from))
	}

	(*m).Merge(AnyMapFrom(from))
	return nil
}

func (m AnyMap) Map() Map {
	to := make(Map, len(m))
	for k, v := range m {
		to[fmt.Sprint(k)] = v
	}

	return to
}

func (m AnyMap) Merge(another AnyMap) {
	m.merge("", another)
}

func (m AnyMap) merge(path string, other AnyMap) {
	for key, value := range other {
		currentPath := fmt.Sprintf("%s.%s", path, key)
		if currentValue, ok := m[key]; !ok {
			m[key] = value
			continue
		} else if currentMapValue, ok := currentValue.(AnyMap); ok {
			if mapValue, ok := value.(AnyMap); ok {
				currentMapValue.merge(currentPath, mapValue)
				continue
			}
		} else if _, ok := value.(AnyMap); !ok {
			m[key] = value
			continue
		}

		log.Panicf("%T keys '%s' must have the same type", m, currentPath)
	}
}

func (m AnyMap) SpecifyTypes() {
	for key, value := range m {
		newKey, removeKey := SpecifyType(key)
		//if str, ok := newKey.(string); ok {
		//	lower := strings.ToLower(str)
		//	if str != lower {
		//		newKey = lower
		//		removeKey = true
		//	}
		//}

		updateValue := false
		if child, ok := value.(Map); ok {
			value = AnyMapFrom(child)
			updateValue = true
		} else if child, ok := value.(AnyMap); ok {
			child.SpecifyTypes()
		} else {
			value, updateValue = SpecifyType(value)
		}

		if removeKey {
			delete(m, key)
		}

		if removeKey || updateValue {
			m[newKey] = value
		}
	}
}

// SpecifyType tries to convert the value to most specific primitive type (int64, float64, bool, or string).
// If it fails, it returns the input value as is.
func SpecifyType(v any) (any, bool) {
	value := reflect.ValueOf(v)
	switch {
	case value.CanInt():
		return value.Int(), true
	case value.CanFloat():
		return value.Float(), true
	case value.Kind() == reflect.Bool:
		return value.Bool(), true
	case value.Kind() == reflect.String:
		str := value.String()
		if strings.HasPrefix(str, "+") {
			return str, false
		}

		if v, err := strconv.ParseInt(str, 10, 64); err == nil {
			return v, true
		} else if v, err := strconv.ParseFloat(str, 64); err == nil {
			return v, true
		} else if v, err := strconv.ParseBool(str); err == nil {
			return v, true
		}

		fallthrough
	default:
		return v, false
	}
}
