package app

import (
	"os"
	"strconv"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func CollectConfig(environPrefix string, inputs ...flu.Input) (*flu.ByteBuffer, error) {
	global := make(map[string]interface{})
	for _, input := range inputs {
		buf := new(flu.ByteBuffer)
		if _, err := flu.Copy(input, buf); err != nil {
			return nil, errors.Wrapf(err, "read config %s", input)
		}

		config := make(map[string]interface{})
		data := flu.Bytes(os.ExpandEnv(buf.Unmask().String()))
		if err := flu.DecodeFrom(data, flu.YAML(&config)); err != nil {
			return nil, errors.Wrapf(err, "read expanded config %s", input)
		}

		global = merge(global, config)
	}

	global = merge(global, environ(environPrefix))
	buf := new(flu.ByteBuffer)
	if err := flu.EncodeTo(flu.YAML(global), buf); err != nil {
		return nil, errors.Wrap(err, "encode global config")
	}

	return buf, nil
}

func environ(prefix string) map[string]interface{} {
	m := make(map[string]interface{})
	for _, line := range os.Environ() {
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		line = line[len(prefix):]
		equals := strings.Index(line, "=")
		key, value := line[:equals], line[equals+1:]
		keyTokens := strings.Split(key, "_")
		keyTokensLastIdx := len(keyTokens) - 1
		entry := m
		for i, keyToken := range keyTokens {
			if keyToken == "" {
				break
			}

			keyToken = strings.ToLower(keyToken)
			if i == keyTokensLastIdx {
				if ev, ok := entry[keyToken]; ok {
					if _, ok := ev.(map[string]interface{}); ok {
						logrus.Warnf("discarding env var %s due to type incompatibility", key)
						continue
					}
				}

				var val interface{}
				if v, err := strconv.ParseInt(value, 10, 64); err == nil {
					val = v
				} else if v, err := strconv.ParseFloat(value, 64); err == nil {
					val = v
				} else if v, err := strconv.ParseBool(value); err == nil {
					val = v
				} else {
					val = value
				}

				entry[keyToken] = val
			} else {
				var mev map[string]interface{}
				if ev, ok := entry[keyToken]; ok {
					if mev, ok = ev.(map[string]interface{}); !ok {
						logrus.Warnf("overriding parent as object for env var %s", key)
						mev = make(map[string]interface{})
						entry[keyToken] = mev
					}
				} else {
					mev = make(map[string]interface{})
					entry[keyToken] = mev
				}

				entry = mev
			}
		}
	}

	return m
}

func merge(a, b map[string]interface{}) map[string]interface{} {
	for k, v := range b {
		if av, ok := a[k]; !ok {
			a[k] = v
			continue
		} else if mav, ok := av.(map[string]interface{}); ok {
			if mv, ok := v.(map[string]interface{}); ok {
				a[k] = merge(mav, mv)
				continue
			}
		} else if _, ok := v.(map[string]interface{}); !ok {
			a[k] = v
			continue
		}

		logrus.Fatalf("configuration keys %s must have the same type", k)
	}

	return a
}
