package redditsave

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu/apfel"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

var URL = "https://redditsave.com"

type Config struct {
	RefreshEvery flu.Duration `yaml:"refreshEvery,omitempty" doc:"Cookie refresh interval" default:"20m"`
}

type Context interface {
	RedditsaveConfig() Config
}

type Client[C Context] struct {
	*client
}

func (c *Client[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	config := app.Config().RedditsaveConfig()
	return c.Standalone(ctx, app, config.RefreshEvery.Value)
}

func (c *Client[C]) Standalone(ctx context.Context, clock syncf.Clock, refreshEvery time.Duration) error {
	c.client = &client{
		client:       new(http.Client),
		clock:        clock,
		refreshEvery: refreshEvery,
	}

	return nil
}

type client struct {
	client       *http.Client
	clock        syncf.Clock
	refreshEvery time.Duration

	lastRefresh time.Time
	mu          syncf.RWMutex
}

func (c *client) String() string {
	return "redditsave.client"
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	logf.Get(c).Resultf(req.Context(), logf.Trace, logf.Warn, "%s => %v", &httpf.RequestBuilder{Request: req}, err)
	return resp, err
}

func (c *client) ResolveURL(ctx context.Context, url string) (string, error) {
	now := c.clock.Now()
	ctx, cancel := c.mu.Lock(ctx)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	defer cancel()

	if now.Sub(c.lastRefresh) <= c.refreshEvery {
		defer cancel()
		var resp resolveResponse
		err := httpf.GET(URL+"/info").
			Query("url", url).
			Exchange(ctx, c).
			CheckStatus(http.StatusOK).
			Handle(&resp).
			Error()
		return resp.url, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", errors.Wrap(err, "create new cookie jar")
	}

	c.client.Jar = jar
	err = httpf.GET(URL).
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		Error()
	logf.Get(c).Resultf(ctx, logf.Debug, logf.Error, "refresh cookie: %v", err)
	if err != nil {
		return "", errors.Wrapf(err, "get [%s] to refresh cookie", URL)
	}

	c.lastRefresh = now
	return c.ResolveURL(ctx, url)
}
