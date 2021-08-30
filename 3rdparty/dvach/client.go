package dvach

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
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

type Client fluhttp.Client

func NewClient(httpClient *fluhttp.Client, usercode string) *Client {
	if httpClient == nil {
		httpClient = fluhttp.NewClient(nil)
	}

	return (*Client)(httpClient.
		SetCookies(Host, cookies(usercode, "/")...).
		SetCookies(Host, cookies(usercode, "/makaba")...).
		AcceptStatus(http.StatusOK))
}

func (c *Client) Unmask() *fluhttp.Client {
	return (*fluhttp.Client)(c)
}

func (c *Client) GetCatalog(ctx context.Context, board string) (*Catalog, error) {
	catalog := new(Catalog)
	err := c.Unmask().GET(Host + "/" + board + "/catalog_num.json").
		Context(ctx).
		Execute().
		DecodeBody(newResponse(catalog)).
		Error
	if err != nil {
		return nil, err
	}
	return catalog, catalog.init(board)
}

func (c *Client) GetThread(ctx context.Context, board string, num int, offset int) ([]Post, error) {
	if offset <= 0 {
		offset = num
	}
	thread := make([]Post, 0)
	err := c.Unmask().GET(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_thread").
		QueryParam("board", board).
		QueryParam("thread", strconv.Itoa(num)).
		QueryParam("num", strconv.Itoa(offset)).
		Context(ctx).
		Execute().
		DecodeBody(newResponse(&thread)).
		Error
	if err != nil {
		return nil, err
	}
	return thread, Posts(thread).init(board)
}

func (c *Client) GetPost(ctx context.Context, board string, num int) (Post, error) {
	posts := make([]Post, 0)
	err := c.Unmask().GET(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_post").
		QueryParam("board", board).
		QueryParam("post", strconv.Itoa(num)).
		Context(ctx).
		Execute().
		DecodeBody(newResponse(&posts)).
		Error
	if err != nil {
		return Post{}, err
	}
	if len(posts) > 0 {
		return posts[0], (&posts[0]).init(board)
	}
	return Post{}, ErrNotFound
}

func (c *Client) DownloadFile(ctx context.Context, file *File, out flu.Output) error {
	return c.Unmask().GET(Host + file.Path).
		Context(ctx).
		Execute().
		DecodeBodyTo(out).
		Error
}

func (c *Client) GetBoards(ctx context.Context) ([]Board, error) {
	boardMap := make(map[string][]Board)
	err := c.Unmask().GET(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_boards").
		Context(ctx).
		Execute().
		DecodeBody(newResponse(&boardMap)).
		Error
	if err != nil {
		return nil, err
	}
	boards := make([]Board, 0)
	for _, boardMapValue := range boardMap {
		boards = append(boards, boardMapValue...)
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
