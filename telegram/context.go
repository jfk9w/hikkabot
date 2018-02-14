package telegram

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type context struct {
	client *http.Client
	token  string
}

func (ctx *context) path(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", Endpoint, ctx.token, method)
}

func (ctx *context) request(req Request) (*Response, error) {
	r, err := ctx.client.PostForm(ctx.path(req.Method()), req.Parameters())
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	resp := new(Response)
	err = json.Unmarshal(data, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (ctx *context) retry(req Request, retries int) (*Response, error) {
	var (
		resp *Response
		err  error
	)

	for {
		resp, err = ctx.request(req)
		if err == nil {
			if resp.Parameters != nil {
				timeout := time.Duration(resp.Parameters.RetryAfter)
				if timeout > 0 {
					time.Sleep(timeout * time.Second)
				}
			}

			break
		}

		if retries == 0 {
			break
		}

		retries--
		time.Sleep(time.Second)
	}

	return resp, err
}