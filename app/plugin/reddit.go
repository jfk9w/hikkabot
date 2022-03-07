package plugin

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"

	"hikkabot/3rdparty/reddit"
	"hikkabot/app"
)

type RedditConfig struct {
	Enabled       bool
	reddit.Config `yaml:"-,inline"`
	RefreshEvery  flu.Duration
}

type RedditClient struct {
	ctx   context.Context
	value *reddit.Client
}

func NewRedditClient(ctx context.Context) *RedditClient {
	return &RedditClient{ctx: ctx}
}

func (c *RedditClient) Get(app app.Interface) (*reddit.Client, error) {
	if c.value != nil {
		return c.value, nil
	}

	var globalConfig struct{ Reddit RedditConfig }
	if err := app.GetConfig().As(&globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Reddit
	if !config.Enabled {
		return nil, nil
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	client := reddit.NewClient(app, config.Config, app.GetVersion())
	app.Manage(client)
	c.value = client
	return client, nil
}
