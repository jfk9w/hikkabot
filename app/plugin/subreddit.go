package plugin

import (
	"context"
	"time"

	"hikkabot/3rdparty/viddit"
	"hikkabot/app"
	"hikkabot/core/feed"
	. "hikkabot/ext/vendors/subreddit"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
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

	Pacing struct {
		Stable     flu.Duration
		Base, Min  *float64
		Multiplier *float64
		MinMembers *int64
		MaxBatch   *int64
	}
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

	pacing := config.Pacing
	vendor := &CommandListener{
		Vendor: &Vendor{
			Context: Context{
				Metrics:        metrics.WithPrefix("subreddit"),
				Clock:          app,
				Storage:        storage,
				MediaManager:   mediaManager,
				RedditClient:   redditClient,
				VidditClient:   vidditClient,
				TelegramClient: bot,
			},
			Pacing: Pacing{
				Stable:     pacing.Stable.GetOrDefault(48 * time.Hour),
				Base:       getFloat(pacing.Base, 0.04),
				Min:        getFloat(pacing.Min, 0.01),
				Multiplier: getFloat(pacing.Multiplier, 10.),
				MinMembers: getInt(pacing.MinMembers, 50),
				MaxBatch:   int(getInt(pacing.MaxBatch, 3)),
			},
			CleanDataEvery: config.Data.CleanEvery.GetOrDefault(30 * time.Minute),
			FreshThingTTL:  config.Storage.FreshTTL.GetOrDefault(7 * 24 * time.Hour),
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
	db, err := app.GetDefaultDatabase()
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

func getFloat(value *float64, defaultValue float64) float64 {
	if value != nil {
		return *value
	}

	return defaultValue
}

func getInt(value *int64, defaultValue int64) int64 {
	if value != nil {
		return *value
	}

	return defaultValue
}
