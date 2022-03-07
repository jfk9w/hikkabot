package reddit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	httpf "github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

var (
	Host         = "https://oauth.reddit.com"
	AuthEndpoint = "https://www.reddit.com/api/v1/access_token"
)

type Config struct {
	ClientID, ClientSecret string
	Username, Password     string

	Owner      string
	MaxRetries int
}

type Client struct {
	*http.Client
	config    Config
	clock     flu.Clock
	token     chan string
	expiresAt time.Time
}

func NewClient(clock flu.Clock, config Config, version string) *Client {
	owner := config.Owner
	if owner == "" {
		owner = config.Username
	}

	token := make(chan string, 1)
	token <- ""

	return &Client{
		Client: &http.Client{
			Transport: withUserAgent(
				httpf.NewDefaultTransport(),
				fmt.Sprintf(`hikkabot/%s by /u/%s`, version, owner)),
		},
		config: config,
		clock:  clock,
		token:  token,
	}
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	select {
	case token := <-c.token:
		return token, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Client) done(ctx context.Context, token string) error {
	select {
	case c.token <- token:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) authorize(ctx context.Context) (string, error) {
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
	errUnauthorized    = errors.New("unauthorized")
	errTooManyRequests = errors.New("too many requests")
)

func (c *Client) execute(ctx context.Context, req *httpf.RequestBuilder) *httpf.ExchangeResult {
	token, err := c.getToken(ctx)
	if err != nil {
		return httpf.ExchangeError(err)
	}

	defer func() { _ = c.done(ctx, token) }()
	var resp *httpf.ExchangeResult
	for i := 0; i < c.config.MaxRetries; i++ {
		if token == "" || c.expiresAt.Before(c.clock.Now()) {
			token, err = c.authorize(ctx)
			if err != nil {
				return httpf.ExchangeError(errors.Wrap(err, "authorize"))
			}
		}

		resp = req.Auth(httpf.Bearer(token)).Exchange(ctx, c)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			token = ""
			continue

		case http.StatusTooManyRequests:
			resetValue := resp.Header.Get("X-Ratelimit-Reset")
			reset, err := strconv.Atoi(resetValue)
			if err != nil {
				return httpf.ExchangeError(errors.Wrapf(err, "parse reset header: %s", resetValue))
			}

			resetAfter := time.Duration(reset) * time.Second
			select {
			case <-ctx.Done():
				return httpf.ExchangeError(ctx.Err())
			case <-time.After(resetAfter):
				continue
			}

		default:
			return resp
		}
	}

	return resp
}

func (c *Client) GetListing(ctx context.Context, subreddit, sort string, limit int) ([]Thing, error) {
	if limit <= 0 {
		limit = 25
	}

	var resp Listing
	if err := c.execute(ctx, httpf.GET(Host+"/r/"+subreddit+"/"+sort).
		Query("limit", strconv.Itoa(limit))).
		CheckStatus(http.StatusOK).
		DecodeBody(&resp).
		Error(); err != nil {
		return nil, errors.Wrap(err, "get listing")
	}

	return resp.Data.Children, nil
}

func (c *Client) GetPosts(ctx context.Context, subreddit string, ids ...string) ([]Thing, error) {
	var resp Listing
	if err := c.execute(ctx, httpf.GET(Host+"/r/"+subreddit+"/api/info").
		Query("id", strings.Join(ids, ","))).
		CheckStatus(http.StatusOK).
		DecodeBody(&resp).
		Error(); err != nil {
		return nil, errors.Wrap(err, "get posts")
	}

	return resp.Data.Children, nil
}

func (c *Client) Subscribe(ctx context.Context, action SubscribeAction, subreddits []string) error {
	return c.execute(ctx, httpf.POST(Host+"/api/subscribe", nil).
		Query("action", string(action)).
		Query("skip_initial_defaults", "true").
		Query("sr_name", strings.Join(subreddits, ","))).
		CheckStatus(http.StatusOK).
		Error()
}

func withUserAgent(rt http.RoundTripper, userAgent string) httpf.RoundTripperFunc {
	return func(req *http.Request) (*http.Response, error) {
		req.Header.Set("User-Agent", userAgent)
		return rt.RoundTrip(req)
	}
}
