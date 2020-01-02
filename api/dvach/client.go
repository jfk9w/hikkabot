package dvach

import (
	"io"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type response struct {
	value interface{}
}

func newResponse(value interface{}) flu.ReaderFrom {
	return &response{value: value}
}

func (r *response) ReadFrom(body io.Reader) (err error) {
	buf := flu.NewBuffer()
	err = flu.Copy(flu.Xable{R: body}, buf)
	if err != nil {
		return errors.Wrap(err, "read body")
	}
	err = flu.Read(buf, flu.JSON(r.value))
	if err == nil {
		return
	}
	err = new(Error)
	if flu.Read(buf, flu.JSON(err)) != nil {
		err = errors.Errorf("failed to decode response: %s", string(buf.Bytes()))
	}
	return
}

type Client struct {
	*flu.Client
}

func NewClient(client *flu.Client, usercode string) *Client {
	if client == nil {
		client = flu.NewClient(nil)
	}
	return &Client{
		Client: client.
			SetCookies(Host, cookies(usercode, "/")...).
			SetCookies(Host, cookies(usercode, "/makaba")...).
			AcceptResponseCodes(http.StatusOK),
	}
}

func (c *Client) GetCatalog(board string) (*Catalog, error) {
	catalog := new(Catalog)
	err := c.GET(Host + "/" + board + "/catalog_num.json").
		Execute().
		Read(newResponse(catalog)).
		Error
	if err != nil {
		return nil, err
	}
	return catalog, catalog.init(board)
}

func (c *Client) GetThread(board string, num int, offset int) ([]Post, error) {
	if offset <= 0 {
		offset = num
	}
	thread := make([]Post, 0)
	err := c.GET(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_thread").
		QueryParam("board", board).
		QueryParam("thread", strconv.Itoa(num)).
		QueryParam("num", strconv.Itoa(offset)).
		Execute().
		Read(newResponse(&thread)).
		Error
	if err != nil {
		return nil, err
	}
	return thread, Posts(thread).init(board)
}

var ErrPostNotFound = errors.New("post not found")

func (c *Client) GetPost(board string, num int) (*Post, error) {
	posts := make([]Post, 0)
	err := c.GET(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_post").
		QueryParam("board", board).
		QueryParam("post", strconv.Itoa(num)).
		Execute().
		Read(newResponse(&posts)).
		Error
	if err != nil {
		return nil, err
	}
	if len(posts) > 0 {
		return &posts[0], (&posts[0]).init(board)
	}
	return nil, ErrPostNotFound
}

func (c *Client) DownloadFile(file *File, out flu.Writable) error {
	return c.GET(Host + file.Path).
		Execute().
		ReadBodyTo(out).
		Error
}

func (c *Client) GetBoards() ([]Board, error) {
	boardMap := make(map[string][]Board)
	err := c.GET(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_boards").
		Execute().
		Read(newResponse(&boardMap)).
		Error
	if err != nil {
		return nil, err
	}
	boards := make([]Board, 0)
	for _, boardMapValue := range boardMap {
		for _, board := range boardMapValue {
			boards = append(boards, board)
		}
	}
	return boards, nil
}

var ErrBoardNotFound = errors.New("board not found")

func (c *Client) GetBoard(id string) (*Board, error) {
	boards, err := c.GetBoards()
	if err != nil {
		return nil, err
	}
	for _, board := range boards {
		if board.ID == id {
			return &board, nil
		}
	}
	return nil, ErrBoardNotFound
}
