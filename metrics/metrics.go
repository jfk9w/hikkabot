package metrics

import (
	"io"
	"sort"
	"strings"
)

type Metrics interface {
	io.Closer
	WithPrefix(prefix string) Metrics
	Counter(name string, labels Labels) Counter
	Gauge(name string, labels Labels) Gauge
}

type Counter interface {
	Inc()
	Add(float64)
}

type Gauge interface {
	Set(float64)
	Inc()
	Dec()
	Add(float64)
	Sub(float64)
}

type Labels map[string]string

func (labels Labels) Keys() []string {
	keys := make([]string, len(labels))
	i := 0
	for key := range labels {
		keys[i] = key
		i++
	}

	return keys
}

func (labels Labels) Values(sep, esc string) string {
	if labels == nil {
		return ""
	}

	keys := labels.Keys()
	sort.Strings(keys)
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = strings.Replace(labels[key], sep, esc, -1)
	}

	return strings.Join(values, ".")
}
