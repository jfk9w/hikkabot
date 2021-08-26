package app

import (
	"context"
	"fmt"
	"io"
	"path"
	"runtime"
	"time"

	"github.com/jfk9w/hikkabot/core/media"

	"github.com/jfk9w/hikkabot/util"

	"github.com/jfk9w/hikkabot/core/access"
	"github.com/jfk9w/hikkabot/core/listener"

	executor "github.com/jfk9w/hikkabot/core/executor"

	fluhttp "github.com/jfk9w-go/flu/http"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/core/aggregator"

	"github.com/jfk9w/hikkabot/core/blob"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w/hikkabot/core/feed"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Instance struct {
	GitCommit string
	Clock     flu.Clock

	config           flu.Input
	converterPlugins []ConverterPlugin
	vendorPlugins    []VendorPlugin
	services         []io.Closer

	db           *gorm.DB
	metrics      metrics.Registry
	mediaManager *media.Manager
}

func Create(gitCommit string, clock flu.Clock, config flu.Input) (*Instance, error) {
	app := &Instance{
		GitCommit:        gitCommit,
		Clock:            clock,
		vendorPlugins:    make([]VendorPlugin, 0),
		converterPlugins: make([]ConverterPlugin, 0),
		services:         make([]io.Closer, 0),
	}

	configs, err := flu.ToString(config)
	if err != nil {
		return nil, errors.Wrap(err, "read config to string")
	}

	logrus.Tracef("configuration: %s", configs)
	app.config = flu.Bytes(configs)

	return app, nil
}

func (app *Instance) GetConfig(value interface{}) error {
	return flu.DecodeFrom(app.config, flu.YAML{Value: value})
}

func (app *Instance) Manage(service io.Closer) {
	app.services = append(app.services, service)
	logrus.WithField("service", fmt.Sprintf("%T", service)).Infof("init ok")
}

func (app *Instance) Close() error {
	for i := len(app.services); i > 0; i-- {
		service := app.services[i-1]
		log := logrus.WithField("service", fmt.Sprintf("%T", service))
		if err := service.Close(); err != nil {
			log.Warnf("close: %s", err)
		} else {
			log.Infof("closed")
		}
	}

	return nil
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

func (app *Instance) GetMetricsRegistry(ctx context.Context) (metrics.Registry, error) {
	if app.metrics != nil {
		return app.metrics, nil
	}

	config := new(struct {
		Prometheus struct{ Address string }
		Graphite   struct {
			Address    string
			FlushEvery flu.Duration
		}
	})

	if err := app.GetConfig(config); err != nil {
		return nil, err
	}

	var registry metrics.Registry
	if address := config.Prometheus.Address; address != "" {
		collectors := []prometheus.Collector{
			collectors.NewBuildInfoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
		}

		listener := metrics.NewPrometheusListener(address).MustRegister(collectors...)
		app.Manage(util.CloserFunc(func() error { return listener.Close(ctx) }))
		registry = listener
	} else if address := config.Graphite.Address; address != "" {
		client := &metrics.GraphiteClient{
			Address: address,
			HGBF:    ".2%f",
			Metrics: make(map[string]metrics.GraphiteMetric),
		}

		client.FlushInBackground(ctx, config.Graphite.FlushEvery.GetOrDefault(time.Minute))
		app.Manage(client)
		registry = client
	} else {
		registry = metrics.DummyRegistry{Log: true}
	}

	app.metrics = registry.WithPrefix("app")
	return app.metrics, nil
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

	manager := &media.Manager{
		Context: &media.Context{
			Clock:    app.Clock,
			Storage:  storage,
			Registry: metrics.WithPrefix("media"),
			Deduplicator: &media.Deduplicator{
				Clock:       app.Clock,
				HashStorage: hashStorage,
			},
			HttpClient: fluhttp.NewClient(nil),
			SizeBounds: [2]int64{10 << 10, telegram.Video.AttachMaxSize()},
			Converters: make(map[string]media.Converter),
			Retries:    3,
		},
		RateLimiter: flu.ConcurrencyRateLimiter(1),
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

func (app *Instance) ConfigureLogging() error {
	globalConfig := new(struct {
		Logging struct {
			Level, Format string
			Frame         bool
		}
	})

	if err := app.GetConfig(globalConfig); err != nil {
		return err
	}

	config := globalConfig.Logging
	logLevel := logrus.InfoLevel
	if config.Level != "" {
		var err error
		logLevel, err = logrus.ParseLevel(config.Level)
		if err != nil {
			return err
		}
	}

	logrus.SetLevel(logLevel)

	var formatter logrus.Formatter
	switch config.Format {
	case "json":
		formatter = new(logrus.JSONFormatter)

	case "text":
		textFormatter := &logrus.TextFormatter{
			FullTimestamp:          true,
			PadLevelText:           true,
			DisableLevelTruncation: true,
		}

		if config.Frame {
			textFormatter.CallerPrettyfier = func(frame *runtime.Frame) (function string, file string) {
				fileName := path.Base(frame.File)
				fileDir := path.Base(path.Dir(frame.File))
				return frame.Function, fmt.Sprintf("%s:%d", path.Join(fileDir, fileName), frame.Line)
			}
		}

		formatter = textFormatter

	default:
		return errors.Errorf("invalid logging format: %s", config.Format)
	}

	logrus.SetFormatter(formatter)

	return nil
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
		GitCommit:     app.GitCommit,
	}

	app.Manage(bot.CommandListener(listener))

	cmd := telegram.Command{
		Chat:    &telegram.Chat{ID: supervisor},
		User:    &telegram.User{ID: supervisor},
		Message: new(telegram.Message),
		Key:     "/status",
	}

	return listener.Status(ctx, bot, cmd)
}

func (app *Instance) createAggregator(ctx context.Context,
	bot *telegram.Bot, accessControl *access.DefaultControl) (*aggregator.Default, error) {

	config := new(struct {
		Aggregator struct {
			RefreshEvery flu.Duration
		}
	})

	if err := app.GetConfig(config); err != nil {
		return nil, err
	}

	storage, err := app.createFeedStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create feed storage")
	}

	metrics, err := app.GetMetricsRegistry(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get metrics registry")
	}

	aggregator := &aggregator.Default{
		Context: &aggregator.Context{
			Clock:   app.Clock,
			Storage: storage,
			EventListener: &listener.Event{
				AccessControl: accessControl,
				Registry:      metrics.WithPrefix("event"),
			},
			Client:   bot,
			Interval: config.Aggregator.RefreshEvery.GetOrDefault(time.Minute),
			Vendors:  make(map[string]feed.Vendor),
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
	bot := telegram.NewBot(fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), token)
	if _, err := bot.GetMe(ctx); err != nil {
		return nil, errors.Wrap(err, "get me")
	}

	return bot, nil
}
