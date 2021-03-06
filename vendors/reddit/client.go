package reddit

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

var (
	Host         = "https://oauth.reddit.com"
	AuthEndpoint = "https://www.reddit.com/api/v1/access_token"
	Timeout      = 2 * time.Second
)

type Config struct {
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Owner        string
	MaxRetries   int
}

type Client struct {
	*fluhttp.Client
	tokenTime   time.Time
	rateLimiter flu.RateLimiter
	config      *Config
}

func NewClient(client *fluhttp.Client, config *Config, gitCommit string) *Client {
	if client == nil {
		client = fluhttp.NewClient(nil)
	}

	owner := config.Owner
	if owner == "" {
		owner = config.Username
	}

	return &Client{
		Client: client.
			AcceptStatus(http.StatusOK).
			SetHeader("User-Agent", fmt.Sprintf(`hikkabot/%s by /u/%s`, gitCommit, owner)),
		rateLimiter: flu.IntervalRateLimiter(Timeout),
		config:      config,
	}
}

func (c *Client) refreshToken(ctx context.Context) error {
	if time.Now().Sub(c.tokenTime).Minutes() <= 58 {
		return nil
	}
	resp := new(struct {
		AccessToken string `json:"access_token"`
	})
	if err := c.POST(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", c.config.Username).
		QueryParam("password", c.config.Password).
		Auth(fluhttp.Basic(c.config.ClientID, c.config.ClientSecret)).
		Context(ctx).
		Execute().
		DecodeBody(flu.JSON{resp}).
		Error; err != nil {
		return err
	}
	c.SetHeader("Authorization", "Bearer "+resp.AccessToken)
	c.tokenTime = time.Now()
	log.Printf("[reddit] refreshed access token: %s", resp.AccessToken)
	return nil
}

func (c *Client) GetListing(ctx context.Context, subreddit, sort string, limit int) ([]Thing, error) {
	c.rateLimiter.Start(ctx)
	defer c.rateLimiter.Complete()

	if limit <= 0 {
		limit = 25
	}

	if err := c.refreshToken(ctx); err != nil {
		return nil, errors.Wrap(err, "refresh token")
	}

	resp := new(struct {
		Data struct {
			Children []Thing `json:"children"`
		} `json:"data"`
	})

	if err := c.GET(Host+"/r/"+subreddit+"/"+sort).
		QueryParam("limit", strconv.Itoa(limit)).
		Context(ctx).
		Execute().
		DecodeBody(flu.JSON{Value: resp}).
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
		child.Data.Created = time.Unix(int64(child.Data.CreatedSecs), 0)
	}

	return resp.Data.Children, nil
}
