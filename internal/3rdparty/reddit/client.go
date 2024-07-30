package reddit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

var (
	Host         = "https://oauth.reddit.com"
	AuthEndpoint = "https://www.reddit.com/api/v1/access_token"
)

type Config struct {
	ClientID     string `yaml:"clientId" doc:"See https://github.com/reddit-archive/reddit/wiki/OAuth2-Quick-Start-Example"`
	ClientSecret string `yaml:"clientSecret" doc:"See https://github.com/reddit-archive/reddit/wiki/OAuth2-Quick-Start-Example"`
	Username     string `yaml:"username" doc:"See https://github.com/reddit-archive/reddit/wiki/OAuth2-Quick-Start-Example"`
	Password     string `yaml:"password" doc:"See https://github.com/reddit-archive/reddit/wiki/OAuth2-Quick-Start-Example"`
	Owner        string `yaml:"owner,omitempty" doc:"This value will be used in User-Agent header. If empty, username will be used."`
	MaxRetries   int    `yaml:"maxRetries,omitempty" doc:"Maximum request retries before giving up." default:"3"`
}

type Context interface {
	RedditConfig() Config
}

type Client[C Context] struct {
	*client
}

func (c *Client[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	config := app.Config().RedditConfig()
	owner := config.Owner
	if owner == "" {
		owner = config.Username
	}

	token := make(chan string, 1)
	token <- ""

	c.client = &client{
		client: &http.Client{
			Transport: withUserAgent(
				httpf.NewDefaultTransport(),
				fmt.Sprintf(`hikkabot/%s by /u/%s`, app.Version(), owner)),
		},
		config: config,
		clock:  app,
		token:  token,
	}

	return nil
}

type client struct {
	client    httpf.Client
	config    Config
	clock     syncf.Clock
	token     chan string
	expiresAt time.Time
}

func (c *client) String() string {
	return "reddit.client"
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	logf.Get(c).Resultf(req.Context(), logf.Trace, logf.Warn, "%s => %v", &httpf.RequestBuilder{Request: req}, err)
	return resp, err
}

func (c *client) getToken(ctx context.Context) (string, error) {
	select {
	case token := <-c.token:
		return token, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *client) done(ctx context.Context, token string) error {
	select {
	case c.token <- token:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *client) authorize(ctx context.Context) (string, error) {
	var resp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := httpf.POST(AuthEndpoint, nil).
		Auth(httpf.Basic(c.config.ClientID, c.config.ClientSecret)).
		Query("grant_type", "password").
		Query("username", c.config.Username).
		Query("password", c.config.Password).
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&resp)).
		Error(); err != nil {
		return "", err
	}

	c.expiresAt = c.clock.Now().Add(time.Duration(resp.ExpiresIn) * time.Second).Add(-time.Minute)
	return resp.AccessToken, nil
}

var (
	errUnauthorized = errors.New("unauthorized")
	errRateLimited  = errors.New("rate-limited")
)

func (c *client) execute(ctx context.Context, req *httpf.RequestBuilder, result any) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = c.done(ctx, token) }()
	var resp *httpf.ExchangeResult
	for i := 0; i < c.config.MaxRetries+1; i++ {
		if token == "" || c.expiresAt.Before(c.clock.Now()) {
			token, err = c.authorize(ctx)
			logf.Get(c).Resultf(ctx, logf.Debug, logf.Error, "refresh token: %v", err)
			if err != nil {
				return errors.Wrap(err, "authorize")
			}
		}

		resp = req.Auth(httpf.Bearer(token)).Exchange(ctx, c)

		resp.HandleFunc(func(resp *http.Response) error {
			switch resp.StatusCode {
			case http.StatusOK:
				return nil

			case http.StatusUnauthorized:
				token = ""
				return errUnauthorized

			case http.StatusTooManyRequests:
				resetValue := resp.Header.Get("X-Ratelimit-Reset")
				reset, err := strconv.Atoi(resetValue)
				if err != nil {
					return errors.Wrapf(err, "parse reset header: %s", resetValue)
				}

				resetAfter := time.Duration(reset) * time.Second
				logf.Get(c).Warnf(ctx, "request overflow, sleeping for %s", resetAfter)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(resetAfter):
					return errRateLimited
				}

			default:
				return nil
			}
		})

		if result != nil {
			resp.DecodeBody(flu.JSON(result))
		}

		err = resp.Error()
		switch err {
		case nil:
			return nil
		case errUnauthorized, errRateLimited:
			continue
		default:
			return err
		}
	}

	return err
}

func (c *client) GetListing(ctx context.Context, subreddit, sort string, limit int) ([]Thing, error) {
	if limit <= 0 {
		limit = 25
	}

	var resp Listing
	if err := c.execute(ctx, httpf.GET(Host+"/r/"+subreddit+"/"+sort).
		Query("limit", strconv.Itoa(limit)),
		&resp); err != nil {
		return nil, errors.Wrap(err, "get listing")
	}

	return resp.Data.Children, nil
}

func (c *client) GetPosts(ctx context.Context, subreddit string, ids ...string) ([]Thing, error) {
	var resp Listing
	if err := c.execute(ctx, httpf.GET(Host+"/r/"+subreddit+"/api/info").
		Query("id", strings.Join(ids, ",")),
		&resp); err != nil {
		return nil, errors.Wrap(err, "get posts")
	}

	return resp.Data.Children, nil
}

func (c *client) Subscribe(ctx context.Context, action SubscribeAction, subreddits []string) error {
	return c.execute(ctx, httpf.POST(Host+"/api/subscribe", nil).
		Query("action", string(action)).
		Query("skip_initial_defaults", "true").
		Query("sr_name", strings.Join(subreddits, ",")),
		nil)
}

func withUserAgent(rt http.RoundTripper, userAgent string) httpf.RoundTripperFunc {
	return func(req *http.Request) (*http.Response, error) {
		req.Header.Set("User-Agent", userAgent)
		return rt.RoundTrip(req)
	}
}
