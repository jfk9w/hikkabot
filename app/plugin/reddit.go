package plugin

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"

	"hikkabot/3rdparty/reddit"
	"hikkabot/app"
)

type RedditConfig struct {
	*reddit.Config `yaml:"-,inline"`
	RefreshEvery   flu.Duration
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

	globalConfig := new(struct{ Reddit *RedditConfig })
	if err := app.GetConfig(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Reddit
	if config == nil {
		return nil, nil
	}

	client := reddit.NewClient(nil, config.Config, app.GetVersion())
	if err := client.RefreshInBackground(c.ctx, config.RefreshEvery.GetOrDefault(55*time.Minute)); err != nil {
		return nil, errors.Wrap(err, "setup")
	}

	app.Manage(client)
	c.value = client
	return client, nil
}
