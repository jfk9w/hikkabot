package dvach

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"

	"github.com/jfk9w-go/flu"
	httpf "github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

type response struct {
	value interface{}
}

func newResponse(value interface{}) flu.DecoderFrom {
	return &response{value: value}
}

func (r *response) DecodeFrom(body io.Reader) (err error) {
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "read body")
	}
	bufr := bytes.NewReader(buf)
	if err = flu.DecodeFrom(flu.IO{R: bufr}, flu.JSON(r.value)); err == nil {
		return
	}
	err = new(Error)
	bufr.Reset(buf)
	if flu.DecodeFrom(flu.IO{R: bufr}, flu.JSON(err)) != nil {
		err = errors.Errorf("failed to decode response: %s", string(buf))
	}
	return
}

type Client http.Client

func NewClient(client *http.Client, usercode string) (*Client, error) {
	if client == nil {
		client = new(http.Client)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "create cookie jar")
	}

	cookieURL := &url.URL{Scheme: "https", Host: Domain}
	jar.SetCookies(cookieURL, cookies(usercode, "/"))
	jar.SetCookies(cookieURL, cookies(usercode, "/makaba"))
	return (*Client)(client), nil
}

func (c *Client) Unmask() *http.Client {
	return (*http.Client)(c)
}

func (c *Client) GetCatalog(ctx context.Context, board string) (*Catalog, error) {
	var catalog Catalog
	if err := httpf.GET(Host+"/"+board+"/catalog_num.json").
		Exchange(ctx, c.Unmask()).
		DecodeBody(newResponse(&catalog)).
		Error(); err != nil {
		return nil, err
	}

	return &catalog, (&catalog).init(board)
}

func (c *Client) GetThread(ctx context.Context, board string, num int, offset int) ([]Post, error) {
	if offset <= 0 {
		offset = num
	}

	var posts Posts
	if err := httpf.GET(Host+"/makaba/mobile.fcgi").
		Query("task", "get_thread").
		Query("board", board).
		Query("thread", strconv.Itoa(num)).
		Query("num", strconv.Itoa(offset)).
		Exchange(ctx, c.Unmask()).
		DecodeBody(newResponse(&posts)).
		Error(); err != nil {
		return nil, err
	}

	return posts, posts.init(board)
}

func (c *Client) GetPost(ctx context.Context, board string, num int) (*Post, error) {
	var posts Posts
	if err := httpf.GET(Host+"/makaba/mobile.fcgi").
		Query("task", "get_post").
		Query("board", board).
		Query("post", strconv.Itoa(num)).
		Exchange(ctx, c.Unmask()).
		DecodeBody(newResponse(&posts)).
		Error(); err != nil {
		return nil, err
	}

	if len(posts) > 0 {
		return &posts[0], (&posts[0]).init(board)
	}

	return nil, ErrNotFound
}

func (c *Client) DownloadFile(ctx context.Context, file *File, out flu.Output) error {
	return httpf.GET(Host+file.Path).
		Exchange(ctx, c.Unmask()).
		DecodeBodyTo(out).
		Error()
}

func (c *Client) GetBoards(ctx context.Context) ([]Board, error) {
	var boardMap map[string][]Board
	if err := httpf.GET(Host+"/makaba/mobile.fcgi").
		Query("task", "get_boards").
		Exchange(ctx, c.Unmask()).
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

func (c *Client) GetBoard(ctx context.Context, id string) (*Board, error) {
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
