package dvach

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
)

type Client struct {
	httpClient *flu.Client
}

func NewClient(httpClient *flu.Client, usercode string) *Client {
	if httpClient == nil {
		httpClient = flu.NewClient(nil)
	}

	return &Client{
		httpClient: httpClient.
			SetCookies(Host, cookies(usercode, "/")...).
			SetCookies(Host, cookies(usercode, "/makaba")...),
	}
}

func defaultBodyProcessor(value interface{}) flu.ReadBytesFunc {
	return func(body []byte) error {
		err := json.Unmarshal(body, value)
		if err != nil {
			apierr := new(Error)
			err := json.Unmarshal(body, apierr)
			if err == nil {
				return apierr
			}
		}

		return err
	}
}

func (c *Client) GetCatalog(board string) (*Catalog, error) {
	catalog := new(Catalog)
	err := c.httpClient.NewRequest().
		GET().
		Resource(Host + "/" + board + "/catalog_num.json").
		Send().
		ReadBytesFunc(defaultBodyProcessor(catalog)).
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
	err := c.httpClient.NewRequest().
		GET().
		Resource(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_thread").
		QueryParam("board", board).
		QueryParam("thread", strconv.Itoa(num)).
		QueryParam("num", strconv.Itoa(offset)).
		Send().
		ReadBytesFunc(defaultBodyProcessor(&thread)).
		Error

	if err != nil {
		return nil, err
	}

	return thread, Posts(thread).init(board)
}

var ErrPostNotFound = errors.New("post not found")

func (c *Client) GetPost(board string, num int) (*Post, error) {
	posts := make([]Post, 0)
	err := c.httpClient.NewRequest().
		GET().
		Resource(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_post").
		QueryParam("board", board).
		QueryParam("post", strconv.Itoa(num)).
		Send().
		ReadBytesFunc(defaultBodyProcessor(&posts)).
		Error

	if err != nil {
		return nil, err
	}

	if len(posts) > 0 {
		return &posts[0], (&posts[0]).init(board)
	}

	return nil, ErrPostNotFound
}

func (c *Client) DownloadFile(file *File, resource flu.WriteResource) error {
	return c.httpClient.NewRequest().
		GET().
		Resource(Host + file.Path).
		Send().
		CheckStatusCode(http.StatusOK).
		ReadResource(resource).
		Error
}

func (c *Client) GetBoards() ([]Board, error) {
	bmap := make(map[string][]Board)
	err := c.httpClient.NewRequest().
		GET().
		Resource(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_boards").
		Send().
		ReadBytesFunc(defaultBodyProcessor(&bmap)).
		Error

	if err != nil {
		return nil, err
	}

	boards := make([]Board, 0)
	for _, barr := range bmap {
		for _, board := range barr {
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
