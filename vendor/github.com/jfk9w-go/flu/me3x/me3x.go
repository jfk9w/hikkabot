// Package me3x package contains abstractions for reporting metrics to Graphite or Prometheus.
package me3x

import (
	"fmt"
)

// Registry is an interface representing a named metric registry.
type Registry interface {

	// WithPrefix returns a copy of this Registry with the new sub-prefix
	// which will be applied to all of new Registry's metrics.
	WithPrefix(prefix string) Registry

	// Counter returns a Counter instance.
	// Labels are used depending on implementation.
	Counter(name string, labels Labels) Counter

	// Gauge returns a Gauge instance.
	// Labels are used depending on implementation.
	Gauge(name string, labels Labels) Gauge

	// Histogram returns a Histogram instance.
	// Labels are used depending on implementation.
	Histogram(name string, labels Labels, buckets []float64) Histogram
}

// Counter is a simple metric which can be incremented.
type Counter interface {

	// Inc increments counter by 1.
	Inc()

	// Add adds delta to the counter.
	Add(delta float64)
}

// Gauge is a metric which value can be set.
type Gauge interface {

	// Set sets the current value.
	Set(value float64)

	// Inc increments current value by 1.
	Inc()

	// Dec decrements current value by 1.
	Dec()

	// Add adds delta to the current value.
	Add(delta float64)

	// Sub subtracts delta from the current value.
	Sub(delta float64)
}

// Histogram is a histogram metric.
type Histogram interface {

	// Observe registers a value within this Histogram.
	Observe(value float64)
}

// ToString returns a string description of a provided value making use of Labeled.
func ToString(value interface{}) string {
	desc := fmt.Sprintf("%T", value)
	if labeled, ok := value.(Labeled); ok {
		desc += labeled.Labels().String()
	}

	return desc
}
