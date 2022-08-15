package dvach

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu/apfel"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
)

type Config struct {
	Usercode string `yaml:"usercode,omitempty" doc:"Auth cookie set for 2ch.hk / and /makaba paths. You can get it from your browser. Required to access hidden boards."`
}

type Context interface {
	DvachConfig() Config
}

type Client[C Context] struct {
	client httpf.Client
}

func (c Client[C]) String() string {
	return "dvach.client"
}

func (c *Client[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	config := app.Config().DvachConfig()
	return c.Standalone(ctx, config)
}

func (c *Client[C]) Standalone(ctx context.Context, config Config) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return errors.Wrap(err, "create cookie jar")
	}

	if config.Usercode != "" {
		cookieURL := &url.URL{Scheme: "https", Host: Domain}
		jar.SetCookies(cookieURL, cookies(config.Usercode, "/"))
		jar.SetCookies(cookieURL, cookies(config.Usercode, "/makaba"))
	} else {
		logf.Get(c).Warnf(ctx, "dvach usercode is empty – hidden boards will be unavailable")
	}

	c.client = &http.Client{Jar: jar}

	return nil
}

func (c *Client[C]) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	logf.Get(c).Resultf(req.Context(), logf.Trace, logf.Warn, "%s => %v", &httpf.RequestBuilder{Request: req}, err)
	return resp, err
}

func (c *Client[C]) GetCatalog(ctx context.Context, board string) (*Catalog, error) {
	var catalog Catalog
	if err := httpf.GET(Host+"/"+board+"/catalog_num.json").
		Exchange(ctx, c).
		DecodeBody(flu.JSON(&catalog)).
		Error(); err != nil {
		return nil, err
	}

	return &catalog, (&catalog).init(board)
}

func (c *Client[C]) GetThread(ctx context.Context, board string, num int, offset int) ([]Post, error) {
	if offset <= 0 {
		offset = num
	}

	var resp struct {
		Posts Posts  `json:"posts,omitempty"`
		Error *Error `json:"error,omitempty"`
	}

	url := fmt.Sprintf("%s/api/mobile/v2/after/%s/%d/%d", Host, board, num, offset)
	if err := httpf.GET(url).
		Exchange(ctx, c).
		DecodeBody(flu.JSON(&resp)).
		Error(); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp.Posts, resp.Posts.init(board)
}

func (c *Client[C]) GetPost(ctx context.Context, board string, num int) (*Post, error) {
	posts, err := c.GetThread(ctx, board, num, num)
	if err != nil {
		return nil, err
	}

	return &posts[0], nil
}

func (c *Client[C]) GetBoards(ctx context.Context) ([]Board, error) {
	var boardMap map[string][]Board
	if err := httpf.GET(Host+"/makaba/mobile.fcgi").
		Query("task", "get_boards").
		Exchange(ctx, c).
		DecodeBody(flu.JSON(&boardMap)).
		Error(); err != nil {
		return nil, err
	}

	var boards []Board
	for _, value := range boardMap {
		boards = append(boards, value...)
	}

	return boards, nil
}

func (c *Client[C]) GetBoard(ctx context.Context, id string) (*Board, error) {
	boards, err := c.GetBoards(ctx)
	if err != nil {
		return nil, err
	}

	for _, board := range boards {
		if board.ID == id {
			return &board, nil
		}
	}

	return nil, ErrNotFound
}
