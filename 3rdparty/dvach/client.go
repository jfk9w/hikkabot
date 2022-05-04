package dvach

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"

	"github.com/jfk9w-go/flu/logf"

	"github.com/jfk9w-go/flu/apfel"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
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
		DecodeBody(newResponse(&catalog)).
		Error(); err != nil {
		return nil, err
	}

	return &catalog, (&catalog).init(board)
}

func (c *Client[C]) GetThread(ctx context.Context, board string, num int, offset int) ([]Post, error) {
	if offset <= 0 {
		offset = num
	}

	var posts Posts
	if err := httpf.GET(Host+"/makaba/mobile.fcgi").
		Query("task", "get_thread").
		Query("board", board).
		Query("thread", strconv.Itoa(num)).
		Query("num", strconv.Itoa(offset)).
		Exchange(ctx, c).
		DecodeBody(newResponse(&posts)).
		Error(); err != nil {
		return nil, err
	}

	return posts, posts.init(board)
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
		DecodeBody(newResponse(&boardMap)).
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

type response struct {
	value interface{}
}

func newResponse(value interface{}) flu.DecoderFrom {
	return &response{value: value}
}

func (r *response) DecodeFrom(body io.Reader) error {
	var buf flu.ByteBuffer
	if _, err := flu.Copy(flu.IO{R: body}, &buf); err != nil {
		return err
	}

	if err := flu.DecodeFrom(buf.Bytes(), flu.JSON(r.value)); err == nil {
		return nil
	}

	var err Error
	if err := flu.DecodeFrom(buf.Bytes(), flu.JSON(&err)); err != nil {
		return errors.Errorf("failed to decode response [%s]", buf.Bytes().String())
	}

	return err
}
