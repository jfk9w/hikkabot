package me3x

import "github.com/jfk9w-go/flu/logf"

type loggingMetric struct {
	name   string
	labels Labels
}

func (m loggingMetric) String() string {
	return m.name + m.labels.String()
}

func (m loggingMetric) log() logf.Interface { return logf.Get(rootLoggerName, "dummy") }

type dummyGauge struct{}

func (dummyGauge) Set(float64) {}
func (dummyGauge) Inc()        {}
func (dummyGauge) Dec()        {}
func (dummyGauge) Add(float64) {}
func (dummyGauge) Sub(float64) {}

type loggingGauge loggingMetric

func (g *loggingGauge) unmask() *loggingMetric {
	return (*loggingMetric)(g)
}

func (g *loggingGauge) log() logf.Interface {
	return g.unmask().log()
}

func (g *loggingGauge) Set(value float64) {
	g.log().Debugf(nil, "gauge %s set %.6f", g.unmask(), value)
}

func (g *loggingGauge) Inc() {
	g.log().Debugf(nil, "gauge %s inc", g.unmask())
}

func (g *loggingGauge) Dec() {
	g.log().Debugf(nil, "gauge %s dec", g.unmask())
}

func (g *loggingGauge) Add(delta float64) {
	g.log().Debugf(nil, "gauge %s add %.6f", g.unmask(), delta)
}

func (g *loggingGauge) Sub(delta float64) {
	g.log().Debugf(nil, "gauge %s sub %.6f", g.unmask(), delta)
}

type dummyHistogram struct{}

func (dummyHistogram) Observe(float64) {}

type loggingHistogram loggingMetric

func (h *loggingHistogram) unmask() *loggingMetric {
	return (*loggingMetric)(h)
}

func (h *loggingHistogram) Observe(value float64) {
	log().Debugf(nil, "histogram %s observe %.6f", h.unmask(), value)
}

// DummyRegistry is a dummy Registry implementation which does not store any metrics.
// Optionally may log calls.
type DummyRegistry struct {
	// Prefix is prefixed used for all metric names.
	Prefix string
	// Log should be set if metric call logging is desired.
	Log bool
}

func (r DummyRegistry) WithPrefix(prefix string) Registry {
	r.Prefix = withPrefix(r.Prefix, prefix, ".")
	return r
}

func (r DummyRegistry) Counter(name string, labels Labels) Counter {
	if r.Log {
		return &loggingGauge{
			name:   withPrefix(r.Prefix, name, "."),
			labels: labels,
		}
	}

	return dummyGauge{}
}

func (r DummyRegistry) Gauge(name string, labels Labels) Gauge {
	if r.Log {
		return &loggingGauge{
			name:   withPrefix(r.Prefix, name, "."),
			labels: labels,
		}
	}

	return dummyGauge{}
}

func (r DummyRegistry) Histogram(name string, labels Labels, _ []float64) Histogram {
	if r.Log {
		return &loggingHistogram{
			name:   withPrefix(r.Prefix, name, "."),
			labels: labels,
		}
	}

	return dummyHistogram{}
}
