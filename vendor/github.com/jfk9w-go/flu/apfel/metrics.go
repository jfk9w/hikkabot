package apfel

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type (
	GraphiteConfig struct {
		Address               string       `yaml:"address" doc:"Graphite server address (host and port). This should be a 'plaintext' address (generally running on port 2003)." example:"graphite.your.host:2003"`
		FlushEvery            flu.Duration `yaml:"flushEvery,omitempty" format:"duration" doc:"Interval to keep between consequent metric flushes." default:"\"1m\""`
		HistogramBucketFormat string       `yaml:"histogramBucketFormat,omitempty" doc:"This is used to format histogram buckets to strings in order to use them in metric path under the hood." default:"%.2f"`
	}

	PrometheusConfig struct {
		Address    string   `yaml:"address" doc:"Prometheus listener address URL." example:"http://localhost:9090/metrics"`
		Collectors []string `yaml:"collectors,omitempty" doc:"Additional Prometheus built-in collectors." enum:"go,build_info,process" default:"[ \"go\", \"build_info\", \"process\" ]"`
	}

	// PrometheusContext is the Prometheus application configuration interface.
	PrometheusContext interface{ PrometheusConfig() PrometheusConfig }
	// GraphiteContext is the Graphite application configuration interface.
	GraphiteContext interface{ GraphiteConfig() GraphiteConfig }
)

// Prometheus is the Prometheus application Mixin.
type Prometheus[C PrometheusContext] struct {
	registry me3x.Registry
}

func (p *Prometheus[C]) String() string {
	return "metrics.prometheus"
}

func (p *Prometheus[C]) Include(ctx context.Context, app MixinApp[C]) error {
	config := app.Config().PrometheusConfig()
	if config.Address == "" {
		logf.Get(p).Warnf(ctx, "address is empty, using dummy")
		p.registry = me3x.DummyRegistry{Log: true}
		return nil
	}

	cs := make([]prometheus.Collector, 0, len(config.Collectors))
	for _, collector := range config.Collectors {
		switch collector {
		case "go":
			cs = append(cs, collectors.NewGoCollector())
		case "process":
			cs = append(cs, collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		case "build_info":
			cs = append(cs, collectors.NewBuildInfoCollector())
		}
	}

	registry := &me3x.PrometheusListener{Address: config.Address}
	registry.MustRegister(cs...)
	p.registry = registry
	logf.Get(p).Infof(ctx, "registered listener")
	return nil

}

// Registry returns the me3x.Registry instance.
func (p *Prometheus[C]) Registry() me3x.Registry {
	return p.registry
}

// Graphite is the Graphite application Mixin.
type Graphite[C GraphiteContext] struct {
	registry me3x.Registry
}

func (g Graphite[C]) String() string {
	return "metrics.graphite"
}

func (g *Graphite[C]) Include(ctx context.Context, app MixinApp[C]) error {
	config := app.Config().GraphiteConfig()
	if config.Address == "" {
		logf.Get(g).Warnf(ctx, "address is empty, using dummy")
		g.registry = me3x.DummyRegistry{Log: true}
		return nil
	}

	client := &me3x.GraphiteClient{
		Address: config.Address,
		Clock:   app,
		HGBF:    config.HistogramBucketFormat,
	}

	client.FlushEvery(config.FlushEvery.Value)
	if err := app.Manage(ctx, client); err != nil {
		return err
	}

	g.registry = client
	logf.Get(g).Infof(ctx, "created graphite registry")
	return nil
}

// Registry returns me3x.Registry instance.
func (g *Graphite[C]) Registry() me3x.Registry {
	return g.registry
}
