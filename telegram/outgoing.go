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

func outgoing(ctx *context) (
	qc chan DeferredRequest, uc chan DeferredRequest, h util.Handle) {

	qc = make(chan DeferredRequest, 1000)
	uc = make(chan DeferredRequest, 20)
	h = util.NewHandle()
	t := time.NewTicker(60 * time.Millisecond)

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
