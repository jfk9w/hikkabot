package telegram

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
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

	resp := new(Response)
	return resp, util.ReadResponse(r, resp)
}

func (ctx *context) retry(req Request, retries int) (*Response, error) {
	var (
		resp *Response
		err  error
	)

	for {
		resp, err = ctx.request(req)
		log.WithFields(log.Fields{
			"req": req,
			"resp": resp,
			"err": err,
			"retry": retries,
		}).Debug("COMM retry")

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