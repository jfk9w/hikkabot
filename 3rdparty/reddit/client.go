package reddit

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	HttpClient *fluhttp.Client
	config     *Config
	token      string
	mu         flu.RWMutex
	work       flu.WaitGroup
	cancel     func()
}

func NewClient(httpClient *fluhttp.Client, config *Config, version string) *Client {
	if httpClient == nil {
		httpClient = fluhttp.NewClient(nil)
	}

	owner := config.Owner
	if owner == "" {
		owner = config.Username
	}

	return &Client{
		HttpClient: httpClient.
			AcceptStatus(http.StatusOK).
			SetHeader("User-Agent", fmt.Sprintf(`hikkabot/%s by /u/%s`, version, owner)),
		config: config,
	}
}

func (c *Client) RefreshInBackground(ctx context.Context, every time.Duration) error {
	if c.cancel != nil {
		return nil
	}

	if err := c.RefreshToken(ctx); err != nil {
		return err
	}

	c.cancel = c.work.Go(ctx, func(ctx context.Context) {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for err := c.RefreshToken(ctx); err != nil; err = c.RefreshToken(ctx) {
					if ctx.Err() != nil {
						return
					}

					logrus.Warnf("reddit token refresh: %s", err)
				}
			}
		}
	})

	return nil
}

func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
		c.work.Wait()
	}

	return nil
}

func (c *Client) Auth() fluhttp.Authorization {
	defer c.mu.RLock().Unlock()
	return fluhttp.Bearer(c.token)
}

func (c *Client) RefreshToken(ctx context.Context) error {
	resp := new(struct {
		AccessToken string `json:"access_token"`
	})

	if err := c.HttpClient.POST(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", c.config.Username).
		QueryParam("password", c.config.Password).
		Auth(fluhttp.Basic(c.config.ClientID, c.config.ClientSecret)).
		Context(ctx).
		Execute().
		DecodeBody(flu.JSON(resp)).
		Error; err != nil {
		return err
	}

	defer c.mu.Lock().Unlock()
	c.token = resp.AccessToken
	return nil
}

func (c *Client) GetListing(ctx context.Context, subreddit, sort string, limit int) ([]Thing, error) {
	if limit <= 0 {
		limit = 25
	}

	resp := new(struct {
		Data struct {
			Children []Thing `json:"children"`
		} `json:"data"`
	})

	if err := c.HttpClient.GET(Host+"/r/"+subreddit+"/"+sort).
		Auth(c.Auth()).
		QueryParam("limit", strconv.Itoa(limit)).
		Context(ctx).
		Execute().
		DecodeBody(flu.JSON(resp)).
		Error; err != nil {
		return nil, errors.Wrap(err, "get listing")
	}

	for i := range resp.Data.Children {
		child := &resp.Data.Children[i]
		var err error
		id := strings.Split(child.Data.Name, "_")[1]
		child.Data.ID, err = strconv.ParseUint(id, 36, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse id: %s", id)
		}
		child.Data.SelfTextHTML = html.UnescapeString(child.Data.SelfTextHTML)
		child.Data.CreatedAt = time.Unix(int64(child.Data.CreatedSecs), 0)
	}

	return resp.Data.Children, nil
}
