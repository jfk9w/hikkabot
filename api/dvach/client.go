package dvach

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type defaultResponseHandler struct {
	value interface{}
}

func newResponseHandler(value interface{}) flu.ResponseHandler {
	return &defaultResponseHandler{value: value}
}

func (h *defaultResponseHandler) Handle(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		return flu.StatusCodeError{r.StatusCode, r.Status}
	}
	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return errors.Wrapf(err, "on reading body")
	}
	err = flu.Read(flu.Bytes(data), flu.JSON(h.value))
	if err == nil {
		return nil
	}
	err = new(Error)
	if flu.Read(flu.Bytes(data), flu.JSON(err)) == nil {
		return err
	}
	return errors.Errorf("failed to decode response: %s", string(data))
}

type Client struct {
	*flu.Client
}

func NewClient(http *flu.Client, usercode string) *Client {
	if http == nil {
		http = flu.NewClient(nil)
	}
	return &Client{
		Client: http.
			SetCookies(Host, cookies(usercode, "/")...).
			SetCookies(Host, cookies(usercode, "/makaba")...),
	}
}

func (c *Client) GetCatalog(board string) (*Catalog, error) {
	catalog := new(Catalog)
	err := c.NewRequest().
		GET().
		Resource(Host + "/" + board + "/catalog_num.json").
		Send().
		HandleResponse(newResponseHandler(catalog)).
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
	err := c.NewRequest().
		GET().
		Resource(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_thread").
		QueryParam("board", board).
		QueryParam("thread", strconv.Itoa(num)).
		QueryParam("num", strconv.Itoa(offset)).
		Send().
		HandleResponse(newResponseHandler(&thread)).
		Error
	if err != nil {
		return nil, err
	}
	return thread, Posts(thread).init(board)
}

var ErrPostNotFound = errors.New("post not found")

func (c *Client) GetPost(board string, num int) (*Post, error) {
	posts := make([]Post, 0)
	err := c.NewRequest().
		GET().
		Resource(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_post").
		QueryParam("board", board).
		QueryParam("post", strconv.Itoa(num)).
		Send().
		HandleResponse(newResponseHandler(&posts)).
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
	return c.NewRequest().
		GET().
		Resource(Host + file.Path).
		Send().
		CheckStatusCode(http.StatusOK).
		ReadBodyTo(out).
		Error
}

func (c *Client) GetBoards() ([]Board, error) {
	boardMap := make(map[string][]Board)
	err := c.NewRequest().
		GET().
		Resource(Host+"/makaba/mobile.fcgi").
		QueryParam("task", "get_boards").
		Send().
		HandleResponse(newResponseHandler(&boardMap)).
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
