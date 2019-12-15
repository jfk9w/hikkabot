package reddit

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
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

type request struct {
	http  *flu.Request
	resp  interface{}
	retry int
	work  sync.WaitGroup
	err   error
}

func newRequest(resp interface{}, http *flu.Request) *request {
	req := new(request)
	req.http = http
	req.resp = resp
	req.work.Add(1)
	return req
}

type Client struct {
	http            *flu.Client
	token           string
	lastTokenUpdate time.Time
	queue           chan *request
	config          *Config
	worker          sync.WaitGroup
}

func NewClient(http *flu.Client, config *Config) *Client {
	if http == nil {
		http = flu.NewClient(nil)
	}
	client := &Client{
		http:   http.AddHeader("User-Agent", config.UserAgent),
		queue:  make(chan *request, 1000),
		config: config,
	}
	if err := client.updateToken(); err != nil {
		panic(err)
	}
	go client.runWorker()
	return client
}

func (c *Client) runWorker() {
	c.worker.Add(1)
	defer c.worker.Done()
	for req := range c.queue {
		err := c.updateToken()
		if err != nil {
			log.Printf("Failed to update reddit token: %v", err)
			err = errors.Wrap(err, "on token update")
		}
		if err == nil {
			err = req.http.
				SetHeader("Authorization", "Bearer "+c.token).
				Send().
				CheckStatusCode(http.StatusOK).
				DecodeBody(flu.JSON(req.resp)).
				Error
		}
		if err != nil && req.retry <= c.config.MaxRetries {
			c.queue <- req
		} else {
			req.err = err
			req.work.Done()
		}
		time.Sleep(Timeout)
	}
}

func (c *Client) submitAndWait(resp interface{}, http *flu.Request) error {
	req := newRequest(resp, http)
	c.queue <- req
	req.work.Wait()
	return req.err
}

func (c *Client) updateToken() error {
	if c.token != "" && time.Now().Sub(c.lastTokenUpdate).Minutes() <= 50 {
		return nil
	}
	tokenResponse := new(struct {
		AccessToken string `json:"access_token"`
	})
	err := c.http.NewRequest().
		POST().
		Resource(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", c.config.Username).
		QueryParam("password", c.config.Password).
		BasicAuth(c.config.ClientID, c.config.ClientSecret).
		Send().
		CheckStatusCode(http.StatusOK).
		DecodeBody(flu.JSON(tokenResponse)).
		Error
	if err != nil {
		return err
	}
	c.token = tokenResponse.AccessToken
	c.lastTokenUpdate = time.Now()
	log.Println("Refreshed reddit access token")
	return nil
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
	err := c.submitAndWait(resp, c.http.NewRequest().
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

func (c *Client) Download(thing *Thing, resource flu.ResourceWriter) error {
	if mediaScanner, ok := mediaScanners[thing.Data.Domain]; ok {
		if thing.Data.ResolvedURL == "" {
			media, err := mediaScanner.Get(c.http, thing.Data.URL)
			if err != nil {
				return errors.Wrap(err, "on Media scan")
			}
			thing.Data.ResolvedURL = media.URL
			thing.Data.Extension = media.Container
		}
	} else {
		return UnsupportedMediaDomainError{thing.Data.Domain}
	}
	return c.http.NewRequest().
		GET().
		Resource(thing.Data.ResolvedURL).
		Send().
		ReadResource(resource).
		Error
}

func (c *Client) Shutdown() {
	close(c.queue)
	c.worker.Wait()
}
