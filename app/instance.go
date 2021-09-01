package app

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/app"
	gormutil "github.com/jfk9w-go/flu/gorm"
	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/core/access"
	"github.com/jfk9w/hikkabot/core/aggregator"
	"github.com/jfk9w/hikkabot/core/blob"
	executor "github.com/jfk9w/hikkabot/core/executor"
	"github.com/jfk9w/hikkabot/core/feed"
	"github.com/jfk9w/hikkabot/core/listener"
	"github.com/jfk9w/hikkabot/core/media"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Instance struct {
	*app.Base
	converterPlugins []ConverterPlugin
	vendorPlugins    []VendorPlugin

	db           *gorm.DB
	mediaManager *media.Manager
}

func Create(version string, clock flu.Clock, config flu.Input) (*Instance, error) {
	base, err := app.New(version, clock, config, flu.YAML)
	if err != nil {
		return nil, err
	}

	return &Instance{
		Base:             base,
		converterPlugins: make([]ConverterPlugin, 0),
		vendorPlugins:    make([]VendorPlugin, 0),
	}, nil
}

func (app *Instance) GetDatabase() (*gorm.DB, error) {
	if app.db != nil {
		return app.db, nil
	}

	config := new(struct{ Database string })
	if err := app.GetConfig(config); err != nil {
		return nil, err
	}

	db, err := gormutil.NewPostgres(config.Database)
	if err != nil {
		return nil, errors.Wrap(err, "open postgres")
	}

	app.Manage((*gormutil.Closer)(db))
	app.db = db
	return db, nil
}

func (app *Instance) GetMediaManager(ctx context.Context) (*media.Manager, error) {
	if app.mediaManager != nil {
		return app.mediaManager, nil
	}

	storage, err := app.createFileStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create file storage")
	}

	hashStorage, err := app.createHashStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create hash storage")
	}

	metrics, err := app.GetMetricsRegistry(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get metrics registry")
	}

	globalConfig := new(struct {
		Media struct {
			Concurrency int
			Retries     int
		}
	})

	if err := app.GetConfig(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Media
	manager := &media.Manager{
		Context: &media.Context{
			Clock:    app,
			Storage:  storage,
			Registry: metrics.WithPrefix("media"),
			Deduplicator: &media.Deduplicator{
				Clock:       app,
				HashStorage: hashStorage,
			},
			HttpClient: fluhttp.NewClient(nil),
			SizeBounds: [2]int64{1 << 10, telegram.Video.AttachMaxSize()},
			Converters: make(map[string]media.Converter),
			Retries:    config.Retries,
		},
		RateLimiter: flu.ConcurrencyRateLimiter(config.Concurrency + 1),
	}

	for _, plugin := range app.converterPlugins {
		id := plugin.ConverterID()
		converter, err := plugin.CreateConverter(ctx, app)
		if err != nil {
			return nil, errors.Wrapf(err, "create %s vendor", id)
		} else if converter == nil {
			continue
		}

		for _, mimeType := range plugin.MIMETypes() {
			if _, ok := manager.Converters[mimeType]; ok {
				return nil, errors.Errorf("duplicate converter for %s", mimeType)
			}

			manager.Converters[mimeType] = converter
		}
	}

	manager.Init(ctx)
	app.Manage(manager)
	app.mediaManager = manager

	return manager, nil
}

func (app *Instance) ApplyVendorPlugins(plugins ...VendorPlugin) {
	app.vendorPlugins = append(app.vendorPlugins, plugins...)
}

func (app *Instance) ApplyConverterPlugins(plugins ...ConverterPlugin) {
	app.converterPlugins = append(app.converterPlugins, plugins...)
}

func (app *Instance) Run(ctx context.Context) error {
	config := new(struct {
		Telegram struct {
			Token      string
			Supervisor telegram.ID
			Aliases    map[string]telegram.ID
		}
	})

	if err := app.GetConfig(config); err != nil {
		return errors.Wrap(err, "get config")
	}

	bot, err := app.createBot(ctx, config.Telegram.Token)
	if err != nil {
		return errors.Wrap(err, "create bot")
	}

	supervisor := config.Telegram.Supervisor
	accessControl := access.NewDefaultControl(supervisor)

	aggregator, err := app.createAggregator(ctx, bot, accessControl)
	if err != nil {
		return errors.Wrap(err, "create aggregator")
	}

	listener := &listener.Command{
		AccessControl: accessControl,
		Aggregator:    aggregator,
		Aliases:       config.Telegram.Aliases,
		Version:       app.GetVersion(),
	}

	app.Manage(bot.CommandListener(listener))

	cmd := &telegram.Command{
		Chat:    &telegram.Chat{ID: supervisor},
		User:    &telegram.User{ID: supervisor},
		Message: new(telegram.Message),
		Key:     "/status",
	}

	return listener.Status(ctx, bot, cmd)
}

func (app *Instance) createAggregator(ctx context.Context,
	bot *telegram.Bot, accessControl *access.DefaultControl) (*aggregator.Default, error) {

	storage, err := app.createFeedStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create feed storage")
	}

	metrics, err := app.GetMetricsRegistry(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get metrics registry")
	}

	globalConfig := new(struct {
		Aggregator struct {
			RefreshEvery flu.Duration
			Preload      int
		}
	})

	if err := app.GetConfig(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Aggregator
	aggregator := &aggregator.Default{
		Context: &aggregator.Context{
			Clock:   app,
			Storage: storage,
			EventListener: &listener.Event{
				AccessControl: accessControl,
				Registry:      metrics.WithPrefix("event"),
			},
			Client:   bot,
			Interval: config.RefreshEvery.GetOrDefault(time.Minute),
			Vendors:  make(map[string]feed.Vendor),
			Preload:  config.Preload,
		},
		Registry: metrics.WithPrefix("aggregator"),
		Executor: app.createTaskExecutor(ctx),
	}

	for _, plugin := range app.vendorPlugins {
		id := plugin.VendorID()
		vendor, err := plugin.CreateVendor(ctx, app)
		if err != nil {
			return nil, errors.Wrapf(err, "create %s vendor", id)
		} else if vendor == nil {
			continue
		}

		if _, ok := aggregator.Vendors[id]; ok {
			return nil, errors.Wrapf(err, "duplicate vendor %s", id)
		}

		aggregator.Vendors[id] = vendor
	}

	if err := aggregator.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "init aggregator")
	}

	return aggregator, nil
}

func (app *Instance) createTaskExecutor(ctx context.Context) *executor.Default {
	executor := executor.NewDefault(ctx)
	app.Manage(executor)
	return executor
}

func (app *Instance) createHashStorage(ctx context.Context) (media.HashStorage, error) {
	db, err := app.GetDatabase()
	if err != nil {
		return nil, errors.Wrap(err, "get database")
	}

	storage := (*media.SQLHashStorage)(db)
	if err := storage.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "init hash storage")
	}

	return storage, nil
}

func (app *Instance) createFileStorage(ctx context.Context) (*blob.FileStorage, error) {
	globalConfig := new(struct {
		Files struct {
			Directory  string
			TTL        flu.Duration
			CleanEvery flu.Duration
		}
	})

	if err := app.GetConfig(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Files
	storage := &blob.FileStorage{
		Directory: config.Directory,
		TTL:       config.TTL.GetOrDefault(10 * time.Minute),
	}

	if err := storage.Init(); err != nil {
		return nil, errors.Wrap(err, "init file storage")
	}

	cleanEvery := config.CleanEvery.GetOrDefault(5 * time.Minute)
	storage.ScheduleMaintenance(ctx, cleanEvery)

	app.Manage(storage)
	return storage, nil
}

func (app *Instance) createFeedStorage(ctx context.Context) (feed.Storage, error) {
	db, err := app.GetDatabase()
	if err != nil {
		return nil, err
	}

	storage := (*feed.SQLStorage)(db)
	if err := storage.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "init feed storage")
	}

	return storage, nil
}

func (app *Instance) createBot(ctx context.Context, token string) (*telegram.Bot, error) {
	bot := telegram.NewBot(ctx, fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), token)
	if _, err := bot.GetMe(ctx); err != nil {
		return nil, errors.Wrap(err, "get me")
	}

	return bot, nil
}
