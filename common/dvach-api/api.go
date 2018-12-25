package dvach

import (
	"strconv"

	"github.com/jfk9w-go/hikkabot/common/httpx"
	"github.com/pkg/errors"
)

type API struct {
	httpx.HTTP
}

func Configure(config Config) *API {
	return &API{httpx.Configure(config.Http)}
}

func (api *API) exec(url string, params httpx.Params, value interface{}) error {
	err := api.Get(url, params, &httpx.JSON{value})
	if err == nil {
		return nil
	}

	switch err := err.(type) {
	case httpx.InvalidFormat:
		apierr := new(Error)
		if err0 := err.UnmarshalJSON(apierr); err0 != nil {
			return err
		}

		return apierr

	default:
		return err
	}
}

func (api *API) Catalog(board Board) (*Catalog, error) {
	var (
		url = Endpoint + "/" + board + "/catalog.json"

		resp = new(Catalog)
		err  error
	)

	err = api.exec(url, nil, resp)
	if err == nil {
		resp.init(board)
	}

	return resp, err
}

func (api *API) Posts(ref Ref, offset Num) ([]*Post, error) {
	if offset <= 0 {
		offset = ref.Num
	}

	var (
		url    = Endpoint + "/makaba/mobile.fcgi"
		params = httpx.Params{
			"task":   {"get_thread"},
			"board":  {ref.Board},
			"thread": {ref.NumString},
			"num":    {strconv.Itoa(offset)},
		}

		resp = make([]*Post, 0)
		err  error
	)

	err = api.exec(url, params, &resp)
	if err == nil {
		for _, post := range resp {
			post.init(ref.Board)
		}
	}

	return resp, err
}

func (api *API) Post(ref Ref) (*Post, error) {
	var (
		url    = Endpoint + "/makaba/mobile.fcgi"
		params = httpx.Params{
			"task":  {"get_post"},
			"board": {ref.Board},
			"post":  {ref.NumString},
		}

		resp = make([]*Post, 0)
	)

	if err := api.exec(url, params, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, errors.New("empty")
	}

	resp[0].init(ref.Board)
	return resp[0], nil
}

func (api *API) Thread(ref Ref) (*Thread, error) {
	var (
		url    = Endpoint + "/makaba/mobile.fcgi"
		params = httpx.Params{
			"task":  {"get_post"},
			"board": {ref.Board},
			"post":  {ref.NumString},
		}

		resp = make([]*Thread, 0)
	)

	if err := api.exec(url, params, &resp); err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, errors.New("empty")
	}

	resp[0].init(ref.Board)
	return resp[0], nil
}
