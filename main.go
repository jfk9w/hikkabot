package main

import (
	"context"
	"os"
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/flu/serde"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w-go/watchdog"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/resolver"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/jfk9w/hikkabot/vendors/dvach"
	"github.com/jfk9w/hikkabot/vendors/reddit"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var GitCommit = "dev"

type Config struct {
	Supervisor telegram.ID
	Datasource struct{ Driver, Conn string }
	Interval   serde.Duration
	Prometheus struct{ Address string }
	Aconvert   struct {
		Servers []int
		Probe   *aconvert.Probe
	}
	Media struct {
		Directory string
		Retries   int
		CURL      string
	}
	Aliases  map[string]telegram.ID
	Telegram struct{ Token string }
	Reddit   reddit.Config
	Dvach    struct{ Usercode string }
	Watchdog watchdog.Config
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := new(Config)
	check(flu.DecodeFrom(flu.File(os.Args[1]), flu.YAML{Value: config}))

	defer watchdog.Run(ctx, config.Watchdog).Complete(ctx)

	store, err := feed.NewSQLStorage(flu.DefaultClock, config.Datasource.Driver, config.Datasource.Conn)
	check(err)
	defer store.Close()

	blobs, err := (&format.FileBlobStorage{
		Directory:     config.Media.Directory,
		TTL:           30 * time.Minute,
		CleanInterval: 10 * time.Minute,
	}).Init()
	check(err)

	metrics := metrics.NewPrometheusListener(config.Prometheus.Address).MustRegister(
		prometheus.NewBuildInfoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector())
	defer metrics.Close(context.Background())

	store.Registry = metrics.WithPrefix("store")
	aconvert := resolver.Aconvert{
		Client: aconvert.NewClient(nil, config.Aconvert.Servers, config.Aconvert.Probe),
	}

	mediam := (&feed.MediaManager{
		DefaultClient: fluhttp.NewTransport().NewClient(),
		SizeBounds:    [2]int64{1 << 10, 75 << 20},
		Storage:       blobs,
		Dedup:         feed.DefaultMediaDedup{Hashes: store},
		RateLimiter:   flu.ConcurrencyRateLimiter(3),
		Metrics:       metrics.WithPrefix("media"),
		Retries:       config.Media.Retries,
		CURL:          config.Media.CURL,
	}).Init(ctx)
	defer mediam.Converter(aconvert).Close()

	executor := feed.NewTaskExecutor()
	defer executor.Close()

	bot := telegram.NewBot(fluhttp.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		NewClient(), config.Telegram.Token)

	aggregator := &feed.Aggregator{
		Executor:          executor,
		Feeds:             store,
		HTMLWriterFactory: feed.TelegramHTML{Sender: bot},
		UpdateInterval:    config.Interval.Duration,
		Metrics:           metrics.WithPrefix("aggregator"),
	}

	initRedditVendor(ctx, metrics, aggregator, mediam, store, config.Reddit)
	initDvachVendors(aggregator, mediam, config.Dvach.Usercode)

	listener, err := (&feed.CommandListener{
		Context:    ctx,
		Aggregator: aggregator,
		Management: feed.NewSupervisorManagement(bot, config.Supervisor),
		Aliases:    config.Aliases,
		GitCommit:  GitCommit,
	}).Init(ctx)
	check(err)
	defer listener.Close()

	defer bot.CommandListener(listener).Close()

	check(listener.Status(ctx, bot, telegram.Command{
		Chat:    &telegram.Chat{ID: config.Supervisor},
		User:    &telegram.User{ID: config.Supervisor},
		Message: new(telegram.Message),
		Key:     "/status"}))

	flu.AwaitSignal()
}

func initRedditVendor(ctx context.Context, metrics metrics.Registry, aggregator *feed.Aggregator, mediam *feed.MediaManager, sqlite3 *feed.SQLStorage, config reddit.Config) error {
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
		Client:       reddit.NewClient(nil, config),
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

func check(err error) error {
	if err != nil {
		panic(err)
	}
	return err
}
