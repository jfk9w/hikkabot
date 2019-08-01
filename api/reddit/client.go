package reddit

import (
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
	httpReq *flu.Request
	resp    interface{}
	retry   int
	done    chan struct{}
	err     error
}

type Client struct {
	httpClient      *flu.Client
	token           string
	lastTokenUpdate time.Time
	queue           chan *request
	config          *Config
	wg              *sync.WaitGroup
}

func NewClient(httpClient *flu.Client, config *Config) *Client {
	if httpClient == nil {
		httpClient = flu.NewClient(nil)
	}

	c := &Client{
		httpClient: httpClient.AddHeader("User-Agent", config.UserAgent),
		queue:      make(chan *request, 1000),
		config:     config,
		wg:         new(sync.WaitGroup),
	}

	if err := c.updateToken(); err != nil {
		panic(err)
	}

	go c.runWorker()
	return c
}

func (c *Client) runWorker() {
	c.wg.Add(1)
	defer c.wg.Done()
	for req := range c.queue {
		err := c.updateToken()
		if err != nil {
			log.Printf("Failed to update reddit token: %v", err)
			err = errors.Wrap(err, "on token update")
		}

		if err == nil {
			err = req.httpReq.
				SetHeader("Authorization", "Bearer "+c.token).
				Send().
				CheckStatusCode(http.StatusOK).
				ReadBody(flu.JSON(req.resp)).
				Error
		}

		if err != nil && req.retry <= c.config.MaxRetries {
			c.queue <- req
		} else {
			req.err = err
			req.done <- struct{}{}
			close(req.done)
		}

		time.Sleep(Timeout)
	}
}

func (c *Client) updateToken() error {
	if c.token != "" && time.Now().Sub(c.lastTokenUpdate).Minutes() <= 50 {
		return nil
	}

	r := new(struct {
		AccessToken string `json:"access_token"`
	})

	err := c.httpClient.NewRequest().
		POST().
		Resource(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", c.config.Username).
		QueryParam("password", c.config.Password).
		BasicAuth(c.config.ClientID, c.config.ClientSecret).
		Send().
		CheckStatusCode(http.StatusOK).
		ReadBody(flu.JSON(r)).
		Error

	if err != nil {
		return err
	}

	c.token = r.AccessToken
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

	req := &request{
		httpReq: c.httpClient.NewRequest().
			GET().
			Resource(Host+"/r/"+subreddit+"/"+string(sort)).
			QueryParam("limit", strconv.Itoa(limit)),
		resp: resp,
		done: make(chan struct{}, 1),
	}

	c.queue <- req
	<-req.done
	if req.err != nil {
		return nil, errors.Wrap(req.err, "on request")
	}

	for i := range resp.Data.Children {
		resp.Data.Children[i].init()
	}

	return resp.Data.Children, nil
}

var (
	ErrInvalidDomain = errors.New("invalid domain")
	allowedDomains   = map[string]struct{}{
		"i.redd.it":   {},
		"i.imgur.com": {},
		"imgur.com":   {},
		"gfycat.com":  {},
	}
)

func (c *Client) Download(thing *Thing, resource flu.WriteResource) error {
	if _, ok := allowedDomains[thing.Data.Domain]; !ok {
		return ErrInvalidDomain
	}

	if thing.Data.ResolvedURL == "" {
		mediaScanner, ok := mediaScanners[thing.Data.Domain]
		if !ok {
			return ErrInvalidDomain
		}

		m, err := mediaScanner(c.httpClient, thing.Data.URL)
		if err != nil {
			return errors.Wrap(err, "on media scan")
		}

		thing.Data.ResolvedURL = m.url
		thing.Data.Extension = m.ext
	}

	return c.httpClient.NewRequest().
		GET().
		Resource(thing.Data.ResolvedURL).
		Send().
		ReadResource(resource).
		Error
}

func (c *Client) Shutdown() {
	close(c.queue)
	c.wg.Wait()
}
