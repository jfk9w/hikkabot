package dvach

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
)

type Client struct {
	http *flu.Client
}

func NewClient(http *flu.Client, usercode string) *Client {
	if http == nil {
		http = flu.NewClient(nil)
	}

	return &Client{
		http: http.
			Cookies(Host, cookies(usercode, "/")...).
			Cookies(Host, cookies(usercode, "/makaba")...),
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

func (c *Client) GetCatalog(boardID string) (*Catalog, error) {
	catalog := new(Catalog)
	err := c.http.NewRequest().
		Get().
		Endpoint(Host + "/" + boardID + "/catalog_num.json").
		Execute().
		ReadBytesFunc(defaultBodyProcessor(catalog)).
		Error

	if err != nil {
		return nil, err
	}

	return catalog, catalog.init(boardID)
}

func (c *Client) GetThread(boardID string, num int, offset int) ([]*Post, error) {
	if offset <= 0 {
		offset = num
	}

	thread := make([]*Post, 0)
	err := c.http.NewRequest().
		Get().
		Endpoint(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_thread").
		QueryParam("board", boardID).
		QueryParam("thread", strconv.Itoa(num)).
		QueryParam("num", strconv.Itoa(offset)).
		Execute().
		ReadBytesFunc(defaultBodyProcessor(&thread)).
		Error

	if err != nil {
		return nil, err
	}

	return thread, posts(thread).init(boardID)
}

var ErrPostNotFound = errors.New("post not found")

func (c *Client) GetPost(boardID string, num int) (*Post, error) {
	posts := make([]*Post, 0)
	err := c.http.NewRequest().
		Get().
		Endpoint(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_post").
		QueryParam("board", boardID).
		QueryParam("post", strconv.Itoa(num)).
		Execute().
		ReadBytesFunc(defaultBodyProcessor(&posts)).
		Error

	if err != nil {
		return nil, err
	}

	if len(posts) > 0 {
		return posts[0], posts[0].init(boardID)
	}

	return nil, ErrPostNotFound
}

func (c *Client) DownloadFile(file *File, resource flu.WriteResource) error {
	return c.http.NewRequest().
		Get().
		Endpoint(Host + file.Path).
		Execute().
		StatusCodes(http.StatusOK).
		ReadResource(resource).
		Error
}

func (c *Client) GetBoards() ([]*Board, error) {
	m := make(map[string][]*Board)
	err := c.http.NewRequest().
		Get().
		Endpoint(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_boards").
		Execute().
		ReadBytesFunc(defaultBodyProcessor(&m)).
		Error

	if err != nil {
		return nil, err
	}

	arr := make([]*Board, 0)
	for _, boards := range m {
		for _, board := range boards {
			arr = append(arr, board)
		}
	}

	return arr, nil
}

var ErrBoardNotFound = errors.New("board not found")

func (c *Client) GetBoard(boardID string) (*Board, error) {
	boards, err := c.GetBoards()
	if err != nil {
		return nil, err
	}

	for _, board := range boards {
		if board.ID == boardID {
			return board, nil
		}
	}

	return nil, ErrBoardNotFound
}
