package metrics

type Metrics interface {
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
