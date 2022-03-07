package plugin

import (
	"context"

	"github.com/pkg/errors"

	"hikkabot/3rdparty/srstats"
	. "hikkabot/app"
	"hikkabot/core/feed"
	. "hikkabot/ext/vendors/srstats"
)

var SubredditStats VendorPlugin = subredditStats{}

type subredditStats struct{}

func (subredditStats) VendorID() string {
	return Name
}

func (subredditStats) CreateVendor(ctx context.Context, app Interface) (feed.Vendor, error) {
	var config struct {
		Telegram TelegramConfig
		Srstats  Config
	}

	if err := app.GetConfig().As(&config); err != nil {
		return nil, err
	}

	if !config.Srstats.Enabled {
		return nil, nil
	}

	bot, err := app.GetBot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get bot")
	}

	events, err := app.GetEventStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get events storage")
	}

	feeds, err := app.GetFeedStorage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get feed storage")
	}

	return &Vendor{
		Clock:    app,
		Telegram: bot,
		Events:   events,
		Feeds:    feeds,
		Stats:    new(srstats.Client),
		Config:   config.Srstats,
		Aliases:  config.Telegram.Aliases,
	}, nil
}
