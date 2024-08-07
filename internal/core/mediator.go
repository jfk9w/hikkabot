package core

import (
	"context"

	"github.com/jfk9w/hikkabot/v4/internal/core/internal/mediator"
	"github.com/jfk9w/hikkabot/v4/internal/feed"
	"github.com/jfk9w/hikkabot/v4/internal/feed/media"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
)

type MediatorConfig struct {
	Concurrency int          `yaml:"concurrency,omitempty" doc:"How many concurrent media downloads to allow." default:"5"`
	Timeout     flu.Duration `yaml:"timeout,omitempty" doc:"If mediation time exceeds timeout, it will be interrupted." default:"10m"`
}

type MediatorService interface {
	feed.Mediator
	RegisterMediaResolver(resolver media.Resolver)
	RegisterMediaConverter(converter media.Converter)
}

type MediatorContext interface {
	apfel.PrometheusContext
	BlobContext
	StorageContext
	MediatorConfig() MediatorConfig
}

type Mediator[C MediatorContext] struct {
	MediatorService
}

func (m Mediator[C]) String() string {
	return mediator.ServiceID
}

func (m *Mediator[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	if m.MediatorService != nil {
		return nil
	}

	var storage Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	var blobs Blobs[C]
	if err := app.Use(ctx, &blobs, false); err != nil {
		return err
	}

	var metrics apfel.Prometheus[C]
	if err := app.Use(ctx, &metrics, false); err != nil {
		return err
	}

	config := app.Config().MediatorConfig()
	mediator := &mediator.Impl{
		Clock:   app,
		Storage: storage,
		Blobs:   blobs,
		Metrics: metrics.Registry().WithPrefix("app_media"),
		Locker:  syncf.Semaphore(app, config.Concurrency, 0),
		Timeout: config.Timeout.Value,
	}

	if err := app.Manage(ctx, mediator); err != nil {
		return err
	}

	m.MediatorService = mediator
	return nil
}

func (m *Mediator[C]) AfterInclude(ctx context.Context, app apfel.MixinApp[C], mixin apfel.Mixin[C]) error {
	if resolver, ok := mixin.(media.Resolver); ok {
		m.RegisterMediaResolver(resolver)
		logf.Get(m).Infof(ctx, "register resolver [%s]: ok", resolver)
	}

	if converter, ok := mixin.(media.Converter); ok {
		m.RegisterMediaConverter(converter)
		logf.Get(m).Infof(ctx, "register converter [%s]: ok", converter)
	}

	return nil
}
