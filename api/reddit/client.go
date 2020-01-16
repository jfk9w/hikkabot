package reddit

import (
	"log"
	"strconv"
	"time"

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
	restraint flu.Restraint
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
		restraint: flu.NewIntervalRestraint(Timeout),
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
	if err := c.POST(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", c.config.Username).
		QueryParam("password", c.config.Password).
		BasicAuth(c.config.ClientID, c.config.ClientSecret).
		Execute().
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
	return req.Execute().ReadBody(flu.JSON(resp)).Error
}

func (c *Client) GetListing(subreddit, sort string, limit int) ([]Thing, error) {
	if limit <= 0 {
		limit = 25
	}
	resp := new(struct {
		Data struct {
			Children []Thing `json:"children"`
		} `json:"data"`
	})
	err := c.execute(resp, c.GET(Host+"/r/"+subreddit+"/"+sort).
		QueryParam("limit", strconv.Itoa(limit)))
	if err != nil {
		return nil, errors.Wrap(err, "on request")
	}
	for i := range resp.Data.Children {
		resp.Data.Children[i].init()
	}
	return resp.Data.Children, nil
}
