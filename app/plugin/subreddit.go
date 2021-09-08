package plugin

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"

	"github.com/jfk9w/hikkabot/3rdparty/viddit"
	"github.com/jfk9w/hikkabot/app"
	"github.com/jfk9w/hikkabot/core/feed"
	. "github.com/jfk9w/hikkabot/ext/vendors/subreddit"
)

type SubredditConfig struct {
	Storage struct {
		FreshTTL   flu.Duration
		CleanEvery flu.Duration
	}

	Viddit struct {
		RefreshEvery flu.Duration
	}

	Data struct {
		CleanEvery flu.Duration
	}

	ConstantPeriod flu.Duration
}

type Subreddit RedditClient

func (p *Subreddit) Unmask() *RedditClient {
	return (*RedditClient)(p)
}

func (p *Subreddit) VendorID() string {
	return "subreddit"
}

func (p *Subreddit) CreateVendor(ctx context.Context, app app.Interface) (feed.Vendor, error) {
	redditClient, err := p.Unmask().Get(app)
	if redditClient == nil {
		return nil, errors.Wrap(err, "create reddit client")
	}

	globalConfig := new(struct{ Subreddit SubredditConfig })
	if err := app.GetConfig(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := &globalConfig.Subreddit

	vidditClient, err := p.createVidditClient(ctx, app, config)
	if err != nil {
		return nil, errors.Wrap(err, "create viddit client")
	}

	storage, err := p.createStorage(ctx, app)
	if err != nil {
		return nil, errors.Wrap(err, "create storage")
	}

	mediaManager, err := app.GetMediaManager(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get media manager")
	}

	eventStorage, err := app.GetEventStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get event storage")
	}

	bot, err := app.GetBot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get bot")
	}

	metrics, err := app.GetMetricsRegistry(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get metrics registry")
	}

	vendor := &CommandListener{
		Vendor: &Vendor{
			Clock:          app,
			Storage:        storage,
			CleanDataEvery: config.Data.CleanEvery.GetOrDefault(30 * time.Minute),
			FreshThingTTL:  config.Storage.FreshTTL.GetOrDefault(7 * 24 * time.Hour),
			RedditClient:   redditClient,
			VidditClient:   vidditClient,
			TelegramClient: bot,
			MediaManager:   mediaManager,
			ConstantPeriod: config.ConstantPeriod.GetOrDefault(3 * 24 * time.Hour),
			Metrics:        metrics.WithPrefix("subreddit"),
		},
		Storage: eventStorage,
	}

	if err := vendor.ScheduleMaintenance(ctx, config.Storage.CleanEvery.GetOrDefault(time.Hour)); err != nil {
		return nil, errors.Wrap(err, "init vendor")
	}

	app.Manage(vendor)
	return vendor, nil
}

func (p Subreddit) createStorage(ctx context.Context, app app.Interface) (Storage, error) {
	db, err := app.GetDatabase()
	if err != nil {
		return nil, err
	}

	storage := (*SQLStorage)(db)
	if err := storage.Init(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

func (p Subreddit) createVidditClient(ctx context.Context, app app.Interface,
	pluginConfig *SubredditConfig) (*viddit.Client, error) {

	config := pluginConfig.Viddit
	client := &viddit.Client{HttpClient: fluhttp.NewClient(nil)}
	if err := client.RefreshInBackground(ctx, config.RefreshEvery.GetOrDefault(20*time.Minute)); err != nil {
		return nil, err
	}

	app.Manage(client)
	return client, nil
}
