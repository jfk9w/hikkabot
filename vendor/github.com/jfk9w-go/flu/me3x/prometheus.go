package me3x

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu/syncf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusListener is a Prometheus-based metric Registry.
type PrometheusListener struct {
	Address  string
	parent   *PrometheusListener
	prefix   string
	server   *http.Server
	registry *prometheus.Registry
	once     sync.Once
}

func (p *PrometheusListener) log() logf.Interface {
	return logf.Get(rootLoggerName, "prometheus")
}

func (p *PrometheusListener) init() {
	u, err := url.Parse(p.Address)
	if err != nil {
		p.log().Panicf(nil, "invalid prometheus address: %v", err)
	}

	p.registry = prometheus.NewRegistry()

	mux := http.NewServeMux()
	mux.Handle(u.Path, promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{}))
	p.server = &http.Server{
		Addr:    u.Host,
		Handler: mux,
	}

	_, _ = syncf.Go(context.Background(), func(ctx context.Context) {
		err := p.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}

		p.log().Resultf(ctx, logf.Debug, logf.Warn, "http server completed with %v", err)
	})
}

func (p *PrometheusListener) MustRegister(cs ...prometheus.Collector) *PrometheusListener {
	if p.parent != nil {
		return p.parent.MustRegister(cs...)
	}

	p.once.Do(p.init)
	p.registry.MustRegister(cs...)
	return p
}

func (p *PrometheusListener) CloseWithContext(ctx context.Context) error {
	if p.server == nil {
		return nil
	}

	return p.server.Shutdown(ctx)
}

func (p *PrometheusListener) Close() error {
	ctx, cancel := syncf.Timeout(10 * time.Second)(context.Background())
	defer cancel()
	return p.CloseWithContext(ctx)
}

func (p *PrometheusListener) WithPrefix(prefix string) Registry {
	return &PrometheusListener{
		parent: p,
		prefix: withPrefix(p.prefix, prefix, "_"),
	}
}

func (p *PrometheusListener) Counter(name string, labels Labels) Counter {
	opts := prometheus.CounterOpts{
		Namespace: p.prefix,
		Name:      name,
	}

	entry := &prometheusEntry[prometheus.Counter, *prometheus.CounterVec]{
		prefix: p.prefix,
		name:   name,
		labels: labels,
		metric: func() prometheus.Counter { return prometheus.NewCounter(opts) },
		vec:    func() *prometheus.CounterVec { return prometheus.NewCounterVec(opts, labels.Names()) },
	}

	return entry.register(p)
}

func (p *PrometheusListener) Gauge(name string, labels Labels) Gauge {
	opts := prometheus.GaugeOpts{
		Namespace: p.prefix,
		Name:      name,
	}

	entry := &prometheusEntry[prometheus.Gauge, *prometheus.GaugeVec]{
		prefix: p.prefix,
		name:   name,
		labels: labels,
		metric: func() prometheus.Gauge { return prometheus.NewGauge(opts) },
		vec:    func() *prometheus.GaugeVec { return prometheus.NewGaugeVec(opts, labels.Names()) },
	}

	return entry.register(p)
}

// this is dumb
type prometheusHistogramVec struct {
	*prometheus.HistogramVec
}

func (v prometheusHistogramVec) With(labels prometheus.Labels) prometheus.Histogram {
	return v.HistogramVec.With(labels).(prometheus.Histogram)
}

func (p *PrometheusListener) Histogram(name string, labels Labels, buckets []float64) Histogram {
	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	opts := prometheus.HistogramOpts{
		Namespace: p.prefix,
		Name:      name,
		Buckets:   buckets,
	}

	entry := &prometheusEntry[prometheus.Histogram, prometheusHistogramVec]{
		prefix: p.prefix,
		name:   name,
		labels: labels,
		metric: func() prometheus.Histogram { return prometheus.NewHistogram(opts) },
		vec: func() prometheusHistogramVec {
			return prometheusHistogramVec{prometheus.NewHistogramVec(opts, labels.Names())}
		},
	}

	return entry.register(p)
}

type prometheusVec[M prometheus.Collector] interface {
	prometheus.Collector
	With(label prometheus.Labels) M
}

type prometheusEntry[M prometheus.Collector, V prometheusVec[M]] struct {
	prefix string
	name   string
	labels Labels
	metric func() M
	vec    func() V
}

func (e *prometheusEntry[M, V]) register(p *PrometheusListener) M {
	if p.parent != nil {
		return e.register(p.parent)
	} else {
		p.once.Do(p.init)
	}

	if len(e.labels) == 0 {
		metric := e.metric()
		var registered prometheus.AlreadyRegisteredError
		if err := p.registry.Register(metric); errors.As(err, &registered) {
			metric = registered.ExistingCollector.(M)
		} else if err != nil {
			logf.Get(p).Panicf(nil, "register metric %s_%s: %v", e.prefix, e.name, err)
		}

		return metric
	}

	vec := e.vec()
	var registered prometheus.AlreadyRegisteredError
	if err := p.registry.Register(vec); errors.As(err, &registered) {
		vec = registered.ExistingCollector.(V)
	} else if err != nil {
		logf.Get(p).Panicf(nil, "register vec %s_%s: %v", e.prefix, e.name, err)
	}

	return vec.With(e.labels.StringMap())
}
