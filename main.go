package main

import (
	"context"
	"os"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/ext/blob"

	"github.com/jfk9w/hikkabot/vendors/dvach/catalog"
	"github.com/jfk9w/hikkabot/vendors/dvach/thread"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	dvach "github.com/jfk9w/hikkabot/3rdparty/dvach"
	reddit "github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/3rdparty/viddit"
	"github.com/jfk9w/hikkabot/feed"
	feedStorage "github.com/jfk9w/hikkabot/feed/storage"
	"github.com/jfk9w/hikkabot/resolver"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/jfk9w/hikkabot/vendors/subreddit"
	subredditStorage "github.com/jfk9w/hikkabot/vendors/subreddit/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// GitCommit denotes the code version holder variable
// and should be substituted with the actual version
// via go build option:
//   -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD)"
var GitCommit = "dev"

type Config struct {
	Media struct {
		*blob.FileStorageConfig `yaml:",inline"`
		Retries                 int
	}

	Database string
	Interval flu.Duration

	// Prometheus contains settings related to metrics reporting.
	Prometheus struct {

		// Address denotes the address used to publish Prometheus metrics endpoint.
		// It should generally be either
		//   "http://localhost:port/path" if you want to publish the endpoint only for the local interface,
		// or
		//   "http://0.0.0.0:port/path' if you want to be able to access it from other computers
		// HTTPS is not supported.
		Address string
	}

	// Aconvert describes the configuration for aconvert.com based conversion service.
	Aconvert struct {

		// Servers is an array of aconvert.com service IDs used for performing conversions.
		// Default value is aconvert.DefaultServers.
		Servers []int

		// Probe is an optional probe used for working servers discovery.
		// If not specified all Servers will be assumed active.
		Probe *aconvert.Probe
	}

	// Telegram describes telegram related settings.
	Telegram struct {

		// Token is the Telegram Bot API token.
		Token string

		// Supervisor is the bot owner user Header used for authorization and feed notifications.
		Supervisor telegram.ID

		// Aliases is a mapping of aliases to chat IDs.
		// These may be used for defining shortcuts of chat names or
		// providing access to private channels or groups.
		Aliases map[string]telegram.ID
	}

	// Reddit describes reddit.com client configuration.
	Reddit *reddit.Config

	// Dvach describes 2ch.hk client configuration.
	Dvach struct {

		// Usercode is set in cookies when performing requests.
		// This used to be required for accessing hidden boards (/e/, /gg/, etc.).
		Usercode string
	}

	// Logging describes logging configuration.
	Logging struct {

		// Format is either "json" or "text". Default is "text".
		Format string

		// Level is the logging level.
		Level string
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := new(Config)
	check(flu.DecodeFrom(flu.File(os.Args[1]), flu.YAML{Value: config}))
	configureLogging(config)

	clock := flu.DefaultClock

	db, err := gormutil.NewPostgres(config.Database)
	check(err)
	defer func() {
		if db, err := db.DB(); err == nil {
			_ = db.Close()
		}
	}()

	blobStorage := &blob.FileStorage{FileStorageConfig: config.Media.FileStorageConfig}

	check(blobStorage.Init(ctx, 10*time.Minute))
	defer blobStorage.Close()

	var metricsRegistry metrics.Registry
	if config.Prometheus.Address != "" {
		metrics := metrics.NewPrometheusListener(config.Prometheus.Address).MustRegister(
			prometheus.NewBuildInfoCollector(),
			prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
			prometheus.NewGoCollector())
		defer metrics.Close(context.Background())
		metricsRegistry = metrics
	} else {
		metricsRegistry = metrics.DummyRegistry{Log: true}
	}

	aconvertResolver := resolver.Aconvert{
		Client: aconvert.NewClient(nil, config.Aconvert.Servers, config.Aconvert.Probe),
	}

	mediaManager := (&feed.MediaManager{
		DefaultClient: fluhttp.NewTransport().NewClient(),
		SizeBounds:    [2]int64{1 << 10, 75 << 20},
		Storage:       blobStorage,
		Dedup: feed.DefaultMediaDedup{
			BlobStorage: (*feedStorage.SQL)(db),
			Clock:       clock,
		},
		RateLimiter: flu.ConcurrencyRateLimiter(3),
		Metrics:     metricsRegistry.WithPrefix("media"),
		Retries:     config.Media.Retries,
	}).Init(ctx)
	defer mediaManager.Converter(aconvertResolver).Close()

	executor := feed.NewTaskExecutor()
	defer executor.Close()

	bot := telegram.NewBot(fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), config.Telegram.Token)

	aggregator := &feed.Aggregator{
		Clock:             clock,
		Executor:          executor,
		Storage:           (*feedStorage.SQL)(db),
		HTMLWriterFactory: feed.TelegramHTML{Sender: bot},
		UpdateInterval:    config.Interval.Unmask(),
		Metrics:           metricsRegistry.WithPrefix("aggregator"),
	}

	dvachClient := dvach.NewClient(nil, config.Dvach.Usercode)

	aggregator.Vendor("2ch/catalog", &catalog.Vendor{
		DvachClient:  dvachClient,
		MediaManager: mediaManager,
	})

	aggregator.Vendor("2ch/thread", &thread.Vendor{
		DvachClient:  dvachClient,
		MediaManager: mediaManager,
		GetTimeout:   20 * time.Second,
	})

	if config.Reddit != nil {
		redditClient := reddit.NewClient(nil, flu.DefaultClock, config.Reddit, GitCommit)
		err := redditClient.RefreshInBackground(ctx, 59*time.Minute)
		check(err)
		defer redditClient.Close()

		vidditClient := &viddit.Client{HttpClient: fluhttp.NewClient(nil)}
		check(vidditClient.RefreshInBackground(ctx, 20*time.Minute))
		defer vidditClient.Close()

		storage := (*subredditStorage.SQL)(db)
		check(storage.Init(ctx))

		subredditVendor := &subreddit.Vendor{
			Clock:         clock,
			Storage:       storage,
			CleanInterval: 30 * time.Minute,
			FreshThingTTL: 7 * 24 * time.Hour,
			RedditClient:  redditClient,
			MediaManager:  mediaManager,
			VidditClient:  vidditClient,
		}

		check(subredditVendor.DeleteStaleThingsInBackground(ctx, time.Hour))
		defer subredditVendor.Close()

		aggregator.Vendor("subreddit", subredditVendor)
	}

	listener, err := (&feed.CommandListener{
		Context:    ctx,
		Aggregator: aggregator,
		Management: feed.NewSupervisorManagement(bot, config.Telegram.Supervisor),
		Aliases:    config.Telegram.Aliases,
		GitCommit:  GitCommit,
	}).Init(ctx)
	check(err)
	defer listener.Close()

	defer bot.CommandListener(listener).Close()

	check(listener.Status(ctx, bot, telegram.Command{
		Chat:    &telegram.Chat{ID: config.Telegram.Supervisor},
		User:    &telegram.User{ID: config.Telegram.Supervisor},
		Message: new(telegram.Message),
		Key:     "/status"}))

	flu.AwaitSignal()
}

func configureLogging(config *Config) {
	logLevel := logrus.InfoLevel
	if config.Logging.Level != "" {
		var err error
		logLevel, err = logrus.ParseLevel(config.Logging.Level)
		check(err)
	}

	logrus.SetLevel(logLevel)

	if config.Logging.Format == "json" {
		logrus.SetFormatter(new(logrus.JSONFormatter))
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:          true,
			PadLevelText:           true,
			DisableLevelTruncation: true,
		})
	}
}

func check(err error) error {
	if err != nil {
		panic(err)
	}
	return err
}
