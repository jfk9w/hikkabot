package reddit

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/lego"
	"github.com/jfk9w-go/lego/pool"
)

var (
	Host         = "https://oauth.reddit.com"
	AuthEndpoint = "https://www.reddit.com/api/v1/access_token"
)

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	UserAgent    string `json:"user_agent"`
}

type Client struct {
	http *flu.Client
	pool pool.Pool
}

func NewClient(http *flu.Client, config *Config) *Client {
	if http == nil {
		http = flu.NewClient(nil)
	}

	return &Client{
		http: http.
			DefaultHeader("User-Agent", config.UserAgent),
		pool: pool.New().Spawn(newWorker(http, config)),
	}
}

func (c *Client) GetListing(subreddit string, sort Sort, limit int) ([]*Thing, error) {
	if limit <= 0 {
		limit = 25
	}

	resp := new(struct {
		Data struct {
			Children []*Thing `json:"children"`
		} `json:"data"`
	})

	ptr := &taskPtr{
		req: c.http.NewRequest().
			Get().
			Endpoint(Host+"/r/"+subreddit+"/"+string(sort)).
			QueryParam("limit", strconv.Itoa(limit)),
		resp: resp,
	}

	err := c.pool.Execute(ptr)
	if err != nil {
		return nil, err
	}

	for _, thing := range resp.Data.Children {
		thing.init()
	}

	return resp.Data.Children, nil
}

var allowedRedditDomains = map[string]struct{}{
	"i.redd.it":   {},
	"i.imgur.com": {},
	"imgur.com":   {},
	"gfycat.com":  {},
}

var ErrInvalidDomain = errors.New("invalid domain")

var genericCanonicalLinkRegexp = regexp.MustCompile(`^.*\.(.*)$`)
var imgurCanonicalLinkRegexp = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*\.(.*?))".*`)

func (c *Client) Download(thing *Thing, resource flu.WriteResource) error {
	if _, ok := allowedRedditDomains[thing.Data.Domain]; !ok {
		return ErrInvalidDomain
	}

	url := thing.Data.URL
	switch thing.Data.Domain {
	case "imgur.com":
		err := c.http.NewRequest().
			Get().
			Endpoint(url).
			Execute().
			ReadBodyFunc(func(body io.Reader) error {
				scanner := bufio.NewScanner(body)
				for scanner.Scan() {
					line := scanner.Text()
					groups := imgurCanonicalLinkRegexp.FindStringSubmatch(line)
					if len(groups) == 4 {
						url = groups[2]
						thing.Data.Extension = groups[3]
						return nil
					}
				}

				return errors.New("unable to find canonical url")
			}).
			Error

		if err != nil {
			return err
		}

	case "gfycat.com":
		url = strings.Replace(thing.Data.URL, "gfycat.com", "giant.gfycat.com", 1) + ".mp4"
		thing.Data.Extension = "mp4"

	default:
		groups := genericCanonicalLinkRegexp.FindStringSubmatch(url)
		if len(groups) == 2 {
			thing.Data.Extension = groups[1]
		} else {
			return errors.New("unable to detect file format")
		}
	}

	thing.Data.URL = url
	return c.http.NewRequest().
		Get().
		Endpoint(url).
		Execute().
		ReadResource(resource).
		Error
}

type worker struct {
	http            *flu.Client
	config          *Config
	token           string
	lastTokenUpdate time.Time
}

var zeroTime = time.Time{}

func newWorker(http *flu.Client, config *Config) *worker {
	w := &worker{
		http:            http,
		config:          config,
		lastTokenUpdate: zeroTime,
	}

	err := w.updateToken()
	lego.Check(err)

	return w
}

func (w *worker) updateToken() error {
	if w.lastTokenUpdate != zeroTime &&
		time.Now().Sub(w.lastTokenUpdate).Minutes() > 50 {
		return nil
	}

	r := new(struct {
		AccessToken string `json:"access_token"`
	})

	err := w.http.NewRequest().
		Post().
		Endpoint(AuthEndpoint).
		QueryParam("grant_type", "password").
		QueryParam("username", w.config.Username).
		QueryParam("password", w.config.Password).
		BasicAuth(w.config.ClientID, w.config.ClientSecret).
		Execute().
		StatusCodes(http.StatusOK).
		ReadBody(flu.JSON(r)).
		Error

	if err != nil {
		return err
	}

	w.token = r.AccessToken
	w.lastTokenUpdate = time.Now()

	return nil
}

func (w *worker) execute(ptr *taskPtr) error {
	err := w.updateToken()
	if err != nil {
		return err
	}

	return ptr.req.
		Header("Authorization", "Bearer "+w.token).
		Execute().
		StatusCodes(http.StatusOK).
		ReadBody(flu.JSON(ptr.resp)).
		Error
}

func (w *worker) Execute(task *pool.Task) {
	ptr := task.Ptr.(*taskPtr)
	err := w.execute(ptr)
	if err != nil && ptr.retry < 3 {
		ptr.retry += 1
		task.Retry()
	} else {
		task.Complete(err)
	}
}

type taskPtr struct {
	req   *flu.Request
	resp  interface{}
	retry int
}
