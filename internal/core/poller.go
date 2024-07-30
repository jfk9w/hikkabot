package core

import (
	"context"

	"github.com/jfk9w/hikkabot/v4/internal/core/internal/poller"
	"github.com/jfk9w/hikkabot/v4/internal/feed"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
)

type PollerService interface {
	feed.Poller
	RegisterVendor(id string, vendor feed.Vendor) error
	RegisterStateListener(listener feed.AfterStateListener)
	RestoreActive(ctx context.Context) error
}

type PollerConfig struct {
	RefreshEvery flu.Duration `yaml:"refreshEvery,omitempty" doc:"Feed update interval." default:"1m"`
	Preload      int          `yaml:"preload,omitempty" doc:"Number of items to preload." default:"5"`
}

type PollerContext interface {
	apfel.PrometheusContext
	tapp.Context
	StorageContext
	PollerConfig() PollerConfig
}

type Poller[C PollerContext] struct {
	PollerService
}

func (p Poller[C]) String() string {
	return poller.ServiceID
}

func (p *Poller[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	if p.PollerService != nil {
		return nil
	}

	var storage Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	var executor TaskExecutor[C]
	if err := app.Use(ctx, &executor, false); err != nil {
		return err
	}

	var metrics apfel.Prometheus[C]
	if err := app.Use(ctx, &metrics, false); err != nil {
		return err
	}

	var bot tapp.Mixin[C]
	if err := app.Use(ctx, &bot, false); err != nil {
		return err
	}

	config := app.Config().PollerConfig()
	p.PollerService = &poller.Impl{
		Clock:    app,
		Storage:  storage,
		Executor: executor,
		Metrics:  metrics.Registry().WithPrefix("app_aggregator"),
		Telegram: bot.Bot(),
		Interval: config.RefreshEvery.Value,
		Preload:  config.Preload,
	}

	return nil
}

func (p *Poller[C]) AfterInclude(ctx context.Context, app apfel.MixinApp[C], mixin apfel.Mixin[C]) error {
	if vendor, ok := mixin.(feed.Vendor); ok {
		err := p.RegisterVendor(mixin.String(), vendor)
		logf.Get(p).Resultf(ctx, logf.Info, logf.Panic, "register vendor [%s]: %v", mixin, err)
	}

	if listener, ok := mixin.(feed.AfterStateListener); ok {
		p.RegisterStateListener(listener)
		logf.Get(p).Infof(ctx, "registered state listener [%s]: ok", mixin)
	}

	return nil
}
