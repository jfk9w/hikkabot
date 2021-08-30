package plugin

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/3rdparty/viddit"
	"github.com/jfk9w/hikkabot/app"
	"github.com/jfk9w/hikkabot/core/feed"
	. "github.com/jfk9w/hikkabot/ext/vendors/subreddit"
	"github.com/pkg/errors"
)

type SubredditConfig struct {
	Client struct {
		*reddit.Config `yaml:"-,inline"`
		RefreshEvery   flu.Duration
	}

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
}

var Subreddit app.VendorPlugin = subreddit{}

type subreddit struct{}

func (p subreddit) VendorID() string {
	return "subreddit"
}

func (p subreddit) CreateVendor(ctx context.Context, app *app.Instance) (feed.Vendor, error) {
	globalConfig := new(struct{ Subreddit *SubredditConfig })
	if err := app.GetConfig(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Subreddit
	if globalConfig.Subreddit == nil {
		return nil, nil
	}

	redditClient, err := p.createRedditClient(ctx, app, config)
	if err != nil {
		return nil, errors.Wrap(err, "create reddit client")
	}

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

	vendor := &Vendor{
		Clock:          app,
		Storage:        storage,
		CleanDataEvery: config.Data.CleanEvery.GetOrDefault(30 * time.Minute),
		FreshThingTTL:  config.Storage.FreshTTL.GetOrDefault(7 * 24 * time.Hour),
		RedditClient:   redditClient,
		VidditClient:   vidditClient,
		MediaManager:   mediaManager,
	}

	if err := vendor.ScheduleMaintenance(ctx, config.Storage.CleanEvery.GetOrDefault(time.Hour)); err != nil {
		return nil, errors.Wrap(err, "init vendor")
	}

	app.Manage(vendor)
	return vendor, nil
}

func (p subreddit) createStorage(ctx context.Context, app *app.Instance) (Storage, error) {
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

func (p subreddit) createVidditClient(ctx context.Context, app *app.Instance,
	pluginConfig *SubredditConfig) (*viddit.Client, error) {

	config := pluginConfig.Viddit
	client := &viddit.Client{HttpClient: fluhttp.NewClient(nil)}
	if err := client.RefreshInBackground(ctx, config.RefreshEvery.GetOrDefault(20*time.Minute)); err != nil {
		return nil, err
	}

	app.Manage(client)
	return client, nil
}

func (p subreddit) createRedditClient(ctx context.Context, app *app.Instance,
	pluginConfig *SubredditConfig) (*reddit.Client, error) {

	config := pluginConfig.Client
	client := reddit.NewClient(nil, config.Config, app.GetVersion())
	if err := client.RefreshInBackground(ctx, config.RefreshEvery.GetOrDefault(59*time.Minute)); err != nil {
		return nil, errors.Wrap(err, "refresh reddit client")
	}

	app.Manage(client)
	return client, nil
}
