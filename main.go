package main

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/flu/serde"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/resolver"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/jfk9w/hikkabot/vendors/dvach"
	"github.com/jfk9w/hikkabot/vendors/reddit"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// GitCommit denotes the code version holder variable
// and should be substituted with the actual version
// via go build option:
//   -ldflags "-X main.GitCommit=$(git rev-parse --short HEAD)"
var GitCommit = "dev"

// Config describes YAML configuration file which is loaded
// when the application is started.
type Config struct {

	// Datasource describes configuration of the datasource
	// used as aggregator backend.
	Datasource struct {

		// Driver should be either "postgres" or "sqlite3".
		// Note that the use of sqlite3 is generally discouraged as a mutex is used
		// for database access synchronization which could lead to lock starvation
		// under some load.
		Driver string

		// Conn represents a connection string passed to the database driver.
		// Examples:
		//   "file::memory:?cache=shared" for sqlite3 would created an in-memory database
		//   "/var/foo/bar/db.sqlite3" for sqlite3 would create and use the database on disk
		//   "postgresql://user:password@host:port/schema" for postgres would connect to a remote instance of PostgreSQL
		Conn string
	}

	// Interval is the amount of time passed between two consecutive subscription update checks
	// for a single feed (chat). Format is the same used in time.ParseDuration ("10s", "2h45m", etc.).
	Interval serde.Duration

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

	// Media is the configuration for media processing.
	Media struct {

		// Directory is a path of directory used for buffering media files.
		Directory string

		// Retries is the amount of retries for performing metadata and content queries.
		Retries int

		// CURL denotes the path to cURL binary for use as a fallback HTTP client. Optional.
		// This may come in handy as I suspect there are issues with Go HTTP/2 implementation.
		CURL string
	}

	// Telegram describes telegram related settings.
	Telegram struct {

		// Token is the Telegram Bot API token.
		Token string

		// Supervisor is the bot owner user ID used for authorization and feed notifications.
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

	store, err := feed.NewSQLStorage(flu.DefaultClock, config.Datasource.Driver, config.Datasource.Conn)
	check(err)
	defer store.Close()

	blobs, err := (&format.FileBlobStorage{
		Directory:     config.Media.Directory,
		TTL:           30 * time.Minute,
		CleanInterval: 10 * time.Minute,
	}).Init()
	check(err)

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

	store.Registry = metricsRegistry.WithPrefix("store")
	aconvert := resolver.Aconvert{
		Client: aconvert.NewClient(nil, config.Aconvert.Servers, config.Aconvert.Probe),
	}

	mediam := (&feed.MediaManager{
		DefaultClient: fluhttp.NewTransport().NewClient(),
		SizeBounds:    [2]int64{1 << 10, 75 << 20},
		Storage:       blobs,
		Dedup: feed.DefaultMediaDedup{
			BlobStorage: store,
		},
		RateLimiter: flu.ConcurrencyRateLimiter(3),
		Metrics:     metricsRegistry.WithPrefix("media"),
		Retries:     config.Media.Retries,
		CURL:        config.Media.CURL,
	}).Init(ctx)
	defer mediam.Converter(aconvert).Close()

	executor := feed.NewTaskExecutor()
	defer executor.Close()

	bot := telegram.NewBot(fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), config.Telegram.Token)

	aggregator := &feed.Aggregator{
		Executor:          executor,
		SubStorage:        store,
		HTMLWriterFactory: feed.TelegramHTML{Sender: bot},
		UpdateInterval:    config.Interval.Duration,
		Metrics:           metricsRegistry.WithPrefix("aggregator"),
	}

	initRedditVendor(ctx, metricsRegistry, aggregator, mediam, store, config.Reddit)
	initDvachVendors(aggregator, mediam, config.Dvach.Usercode)

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

func initRedditVendor(ctx context.Context, metrics metrics.Registry, aggregator *feed.Aggregator, mediam *feed.MediaManager, sqlite3 *feed.SQLStorage, config *reddit.Config) error {
	if config == nil {
		return nil
	}

	store, err := (&reddit.SQLStorage{
		SQLStorage:    sqlite3,
		ThingTTL:      7 * 24 * time.Hour,
		CleanInterval: time.Hour,
	}).Init(ctx)
	if err != nil {
		return errors.Wrap(err, "init reddit store")
	}

	viddit := &common.Viddit{
		Client:        fluhttp.NewClient(nil),
		Clock:         flu.DefaultClock,
		ResetInterval: 20 * time.Minute,
	}

	aggregator.Vendor("subreddit", &reddit.SubredditFeed{
		Client:       reddit.NewClient(nil, config, GitCommit),
		Store:        store,
		MediaManager: mediam,
		Viddit:       viddit,
		Metrics:      metrics.WithPrefix("subreddit"),
	})

	return nil
}

func initDvachVendors(aggregator *feed.Aggregator, mediam *feed.MediaManager, usercode string) {
	client := dvach.NewClient(nil, usercode)

	aggregator.Vendor("2ch/catalog", &dvach.CatalogFeed{
		Client:       client,
		MediaManager: mediam,
	})

	aggregator.Vendor("2ch/thread", &dvach.ThreadFeed{
		Client:       client,
		MediaManager: mediam,
	})
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
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
}

func check(err error) error {
	if err != nil {
		panic(err)
	}
	return err
}
