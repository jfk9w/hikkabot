package reddit

import (
	"fmt"
	"log"
	"strconv"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"
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
	UserAgent    string
	MaxRetries   int
}

type Client struct {
	*flu.Client
	tokenTime time.Time
	restraint telegram.Restraint
	config    Config
}

func NewClient(http *flu.Client, config Config) *Client {
	if http == nil {
		http = flu.NewClient(nil)
	}
	http.AcceptResponseCodes(200).
		SetHeader("User-Agent", config.UserAgent)
	client := &Client{
		Client:    http,
		restraint: telegram.NewIntervalRestraint(Timeout),
		config:    config,
	}
	if err := client.updateToken(); err != nil {
		panic(err)
	}
	return client
}

func (c *Client) updateToken() error {
	resp := new(struct {
		AccessToken string `json:"access_token"`
	})
	if err := c.NewRequest().
		POST().
		Resource(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", c.config.Username).
		QueryParam("password", c.config.Password).
		BasicAuth(c.config.ClientID, c.config.ClientSecret).
		Send().
		ReadBody(flu.JSON(resp)).
		Error; err != nil {
		return err
	}
	c.SetHeader("Authorization", "Bearer "+resp.AccessToken)
	c.tokenTime = time.Now()
	log.Println("Refreshed reddit access token")
	return nil
}

func (c *Client) execute(resp interface{}, req *flu.Request) error {
	c.restraint.Start()
	defer c.restraint.Complete()
	if time.Now().Sub(c.tokenTime).Minutes() > 58 {
		if err := c.updateToken(); err != nil {
			return err
		}
	}
	return req.Send().
		ReadBody(flu.JSON(resp)).
		Error
}

func (c *Client) GetListing(subreddit string, sort Sort, limit int) ([]Thing, error) {
	if limit <= 0 {
		limit = 25
	}
	resp := new(struct {
		Data struct {
			Children []Thing `json:"children"`
		} `json:"data"`
	})
	err := c.execute(resp, c.NewRequest().
		GET().
		Resource(Host+"/r/"+subreddit+"/"+sort).
		QueryParam("limit", strconv.Itoa(limit)))
	if err != nil {
		return nil, errors.Wrap(err, "on request")
	}
	for i := range resp.Data.Children {
		resp.Data.Children[i].init()
	}
	return resp.Data.Children, nil
}

type UnsupportedMediaDomainError struct {
	Domain string
}

func (e UnsupportedMediaDomainError) Error() string {
	return fmt.Sprintf("unsupported Media domain: %s", e.Domain)
}

func (c *Client) Download(thing *Thing, out flu.Writable) error {
	if mediaScanner, ok := mediaScanners[thing.Data.Domain]; ok {
		if thing.Data.ResolvedURL == "" {
			media, err := mediaScanner.Get(c.Client, thing.Data.URL)
			if err != nil {
				return errors.Wrap(err, "on Media scan")
			}
			thing.Data.ResolvedURL = media.URL
			thing.Data.Extension = media.Container
		}
	} else {
		return UnsupportedMediaDomainError{thing.Data.Domain}
	}
	return c.NewRequest().
		GET().
		Resource(thing.Data.ResolvedURL).
		Send().
		ReadBodyTo(out).
		Error
}
