package me3x

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/jfk9w-go/flu"
)

// GraphiteTimeout is the timeout for sending metrics to Graphite.
var GraphiteTimeout = 1 * time.Minute

// GraphiteMetric is a metric which can be sent to Graphite.
type GraphiteMetric interface {

	// Write writes current metric value to Graphite, resetting the value if necessary.
	Write(b *strings.Builder, now string, key string)
}

// GraphiteCounter is a Prometheus counter emulation.
type GraphiteCounter AtomicFloat64

func (c *GraphiteCounter) Inc() {
	c.Add(1)
}

func (c *GraphiteCounter) Add(delta float64) {
	(*AtomicFloat64)(c).Add(delta)
}

func (c *GraphiteCounter) Write(b *strings.Builder, now string, key string) {
	zero := float64(0)
	value := (*AtomicFloat64)(c).Swap(zero)
	if value == zero {
		return
	}

	b.WriteString(fmt.Sprintf("%s %.9f %s\n", key, value, now))
}

// GraphiteGauge is a Prometheus gauge emulation.
type GraphiteGauge AtomicFloat64

func (g *GraphiteGauge) Set(value float64) {
	(*AtomicFloat64)(g).Set(value)
}

func (g *GraphiteGauge) Inc() {
	g.Add(1)
}

func (g *GraphiteGauge) Dec() {
	g.Add(-1)
}

func (g *GraphiteGauge) Add(delta float64) {
	(*AtomicFloat64)(g).Add(delta)
}

func (g *GraphiteGauge) Sub(delta float64) {
	g.Add(-delta)
}

func (g *GraphiteGauge) Write(b *strings.Builder, now string, key string) {
	value := (*AtomicFloat64)(g).Get()
	b.WriteString(fmt.Sprintf("%s %.9f %s\n", key, value, now))
}

// GraphiteHistogram is a Prometheus histogram emulation.
// This is a really primitive emulation.
type GraphiteHistogram struct {
	buckets  []float64
	counters []*GraphiteCounter
	hbf      string
}

func (h GraphiteHistogram) Observe(value float64) {
	idx := len(h.buckets)
	for i, upper := range h.buckets {
		if value < upper {
			idx = i
			break
		}
	}

	h.counters[idx].Inc()
}

func (h GraphiteHistogram) Write(b *strings.Builder, now string, key string) {
	for i, counter := range h.counters {
		bucket := "inf"
		if h.buckets[i] != math.MaxFloat64 {
			bucket = fmt.Sprintf(h.hbf, h.buckets[i])
		}

		counter.Write(b, now, key+"."+strings.Replace(bucket, ".", "_", 1))
	}
}

// GraphiteClient is a Registry implementation for Graphite.
type GraphiteClient struct {

	// Address is the address for sending metrics to.
	// See https://graphite.readthedocs.io/en/latest/feeding-carbon.html.
	Address string

	// Clock is used for getting timestamps when sending data.
	Clock syncf.Clock

	// HGBF is histogram bucket format.
	// It is a string pattern used to transform histogram bucket values into metric name suffix.
	HGBF string

	metrics map[string]GraphiteMetric
	prefix  string
	cancel  context.CancelFunc
	mu      syncf.RWMutex
	parent  *GraphiteClient
}

func (c *GraphiteClient) log() logf.Interface {
	return logf.Get(rootLoggerName, "graphite")
}

// Flush flushes metrics to Graphite.
func (c *GraphiteClient) Flush() error {
	if c.parent != nil {
		return c.parent.Flush()
	}

	var b strings.Builder
	nowstr := strconv.FormatInt(c.Clock.Now().Unix(), 10)

	_, cancel := c.mu.RLock(nil)
	for key, metric := range c.metrics {
		metric.Write(&b, nowstr, key)
	}

	cancel()
	if b.Len() == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), GraphiteTimeout)
	defer cancel()
	data := &flu.Text{Value: b.String()}
	conn := flu.Conn{Context: ctx, Network: "tcp", Address: c.Address}
	return flu.EncodeTo(data, conn)
}

// FlushEvery starts a goroutine which flushes metrics from this GraphiteClient to the Graphite itself.
func (c *GraphiteClient) FlushEvery(interval time.Duration) {
	if c.cancel != nil || c.Address == "" {
		return
	}

	c.cancel = syncf.GoSync(context.Background(), func(ctx context.Context) {
		timer := time.NewTicker(interval)
		defer timer.Stop()
		c.log().Infof(ctx, "will flush metrics every %s", interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				c.flushQuietly()
			}
		}
	})
}

// Close stops the goroutine if it was started
func (c *GraphiteClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	time.Sleep(time.Second)
	return c.Flush()
}

func (c *GraphiteClient) flushQuietly() {
	if err := c.Flush(); err != nil {
		c.log().Printf(nil, "flush metrics error: %v", err)
	}
}

func (c *GraphiteClient) WithPrefix(prefix string) Registry {
	return &GraphiteClient{
		HGBF:   c.HGBF,
		prefix: withPrefix(c.prefix, prefix, "."),
		parent: c,
	}
}

func (c *GraphiteClient) Counter(name string, labels Labels) Counter {
	key := c.makeKey(name, labels)
	entry, ok := getGraphiteEntry[*GraphiteCounter](c, key)
	if !ok {
		entry = createGraphiteEntry(c, key, func() *GraphiteCounter { return new(GraphiteCounter) })
	}

	return entry
}

func (c *GraphiteClient) Gauge(name string, labels Labels) Gauge {
	key := c.makeKey(name, labels)
	entry, ok := getGraphiteEntry[*GraphiteGauge](c, key)
	if !ok {
		entry = createGraphiteEntry(c, key, func() *GraphiteGauge { return new(GraphiteGauge) })
	}

	return entry
}

func (c *GraphiteClient) Histogram(name string, labels Labels, buckets []float64) Histogram {
	key := c.makeKey(name, labels)
	entry, ok := getGraphiteEntry[GraphiteHistogram](c, key)
	if !ok {
		entry = createGraphiteEntry(c, key, func() GraphiteHistogram {
			buckets := append(buckets, math.MaxFloat64)
			counters := make([]*GraphiteCounter, len(buckets))
			for i := range buckets {
				counters[i] = new(GraphiteCounter)
			}

			return GraphiteHistogram{
				buckets:  buckets,
				counters: counters,
				hbf:      c.HGBF,
			}
		})
	}

	return entry
}

func createGraphiteEntry[M GraphiteMetric](c *GraphiteClient, key string, create func() M) M {
	if c.parent != nil {
		return createGraphiteEntry[M](c.parent, key, create)
	}

	_, cancel := c.mu.Lock(nil)
	defer cancel()

	if entry, ok := c.metrics[key]; ok {
		return entry.(M)
	}

	if c.metrics == nil {
		c.metrics = make(map[string]GraphiteMetric)
	}

	entry := create()
	c.metrics[key] = entry
	return entry
}

func getGraphiteEntry[M GraphiteMetric](c *GraphiteClient, key string) (m M, ok bool) {
	if c.parent != nil {
		return getGraphiteEntry[M](c.parent, key)
	}

	_, cancel := c.mu.RLock(nil)
	defer cancel()

	if entry, ok := c.metrics[key]; ok {
		return entry.(M), true
	}

	return
}

func (c *GraphiteClient) makeKey(name string, labels Labels) string {
	prefix := c.prefix
	if prefix != "" {
		prefix += "."
	}

	values := labels.graphitePath(".", "_")
	prefix += values
	if values != "" {
		prefix += "."
	}

	return prefix + name
}
