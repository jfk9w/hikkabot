package telegram

import (
	"encoding/json"
	"time"

	"github.com/jfk9w/hikkabot/util"
)

type Response struct {
	Ok          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Result      json.RawMessage     `json:"result"`
	Parameters  *ResponseParameters `json:"parameters"`
}

type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id"`
	RetryAfter      int   `json:"retry_after"`
}

func (r *Response) Parse(v interface{}) error {
	data, err := r.Result.MarshalJSON()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

type ResponseHandler func(resp *Response, err error)

type DeferredRequest struct {
	request Request
	handler ResponseHandler
}

func outgoing(ctx *context, par int) (
	qc chan DeferredRequest, uc chan DeferredRequest, h util.Handle) {

	qc = make(chan DeferredRequest, 1000)
	uc = make(chan DeferredRequest, 20)
	h = util.NewHandle()
	t := time.NewTicker(time.Duration(60 / par + 1) * time.Millisecond)

	go func() {
		defer func() {
			h.Reply()
			t.Stop()
		}()

		for {
			select {
			case <-t.C:
				select {
				case r := <-uc:
					resp, err := ctx.retry(r.request, 5)
					if r.handler != nil {
						r.handler(resp, err)
					}

				case r := <-qc:
					resp, err := ctx.retry(r.request, 2)
					if r.handler != nil {
						r.handler(resp, err)
					}

				default:
				}

			case <-h.C:
				return
			}
		}
	}()

	return
}

func incoming(ctx *context, req GetUpdatesRequest) (chan Update, util.Handle) {
	c := make(chan Update, 20)
	h := util.NewHandle()
	go func() {
		defer func() {
			close(c)
			h.Reply()
		}()

		for {
			select {
			case <-h.C:
				return

			default:
				resp, err := ctx.request(req)
				if err != nil || !resp.Ok {
					time.Sleep(3 * time.Second)
					continue
				}

				updates := make([]Update, 0)
				err = resp.Parse(&updates)
				if err != nil {
					// logging
					continue
				}

				for _, update := range updates {
					c <- update
					offset := update.ID + 1
					if req.Offset < offset {
						req.Offset = offset
					}
				}
			}
		}
	}()

	return c, h
}
