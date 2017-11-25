package telegram

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
)

// The response contains a JSON object, which always has a Boolean field ‘ok’
// and may have an optional String field ‘description’ with a human-readable
// description of the result. If ‘ok’ equals true, the request was successful
// and the result of the query can be found in the ‘result’ field. In case of
// an unsuccessful request, ‘ok’ equals false and the error is explained
// in the ‘description’. An Integer ‘error_code’ field is also returned,
// but its contents are subject to change in the future.
// Some errors may also have an optional field ‘parameters’
// of the type ResponseParameters, which can help to automatically handle the error.
type Response struct {
	Ok          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Result      json.RawMessage     `json:"result"`
	Parameters  *ResponseParameters `json:"parameters"`
}

// Contains information about why a request was unsuccessful.
type ResponseParameters struct {
	// Optional. The group has been migrated to a supergroup with
	// the specified identifier. This number may be greater than 32 bits
	// and some programming languages may have difficulty/silent defects
	// in interpreting it. But it is smaller than 52 bits,
	// so a signed 64 bit integer or double-precision float type
	// are safe for storing this identifier.
	MigrateToChatID int64 `json:"migrate_to_chat_id"`

	// Optional. In case of exceeding flood control,
	// the number of seconds left to wait before the request can be repeated
	RetryAfter int `json:"retry_after"`
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

const (
	GatewayChoking = "gateway choking"
)

type gateway struct {
	client *http.Client
	token  string

	urgent chan DeferredRequest
	queue  chan DeferredRequest

	halt util.Hook
	done util.Hook
}

func newGateway(client *http.Client, token string) *gateway {
	if client == nil {
		client = new(http.Client)
	}

	return &gateway{
		client: client,
		token:  token,
		urgent: make(chan DeferredRequest, 20),
		queue:  make(chan DeferredRequest, 10000),
		halt:   util.NewHook(),
		done:   util.NewHook(),
	}
}

func (svc *gateway) start() {
	go func() {
		ticker := time.NewTicker(60 * time.Millisecond)
		defer func() {
			svc.done.Send()
			ticker.Stop()
		}()

		for {
			select {
			case <-ticker.C:
				select {
				case r := <-svc.urgent:
					resp, err := svc.retryRequest(r.request, 2)
					if err != nil && err.Error() == GatewayChoking {
						return
					}

					if r.handler != nil {
						r.handler(resp, err)
					}

				case r := <-svc.queue:
					resp, err := svc.retryRequest(r.request, 5)
					if err != nil && err.Error() == GatewayChoking {
						return
					}

					if r.handler != nil {
						r.handler(resp, err)
					}

				default:
				}

			case <-svc.halt:
				return
			}
		}
	}()

	sawmill.Info("gateway started")
}

func (svc *gateway) stop() {
	svc.halt.Send()
	svc.done.Wait()
}

func (svc *gateway) submit(request Request, handler ResponseHandler, urgent bool) {
	sawmill.Debug("gateway submit", sawmill.Fields{
		"request": request,
		"urgent":  urgent,
	})

	var c chan DeferredRequest
	if urgent {
		c = svc.urgent
	} else {
		c = svc.queue
	}

	c <- DeferredRequest{request, handler}
}

func (svc *gateway) retryRequest(request Request, retries int) (*Response, error) {
	var (
		resp *Response
		err  error
	)

	for {
		resp, err = svc.makeRequest(request)
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

func (svc *gateway) makeRequest(request Request) (*Response, error) {
	onSendFailed := func(err error) {
		sawmill.Warning("gateway request", sawmill.Fields{
			"request.Method":     request.Method(),
			"request.Parameters": request.Parameters(),
			"error":              err,
		})
	}

	resp, err := http.PostForm(svc.endpoint(request.Method()), request.Parameters())
	if err != nil {
		onSendFailed(err)
		return nil, err
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		onSendFailed(err)
		return nil, err
	}

	response := new(Response)
	err = json.Unmarshal(data, response)
	if err != nil {
		onSendFailed(err)
		return nil, err
	}

	sawmill.Debug("gateway request", sawmill.Fields{
		"request.Method":       request.Method(),
		"request.Parameters":   request.Parameters(),
		"response.Ok":          response.Ok,
		"response.ErrorCode":   response.ErrorCode,
		"response.Description": response.Description,
		"response.Parameters":  response.Parameters,
	})

	return response, nil
}

func (svc *gateway) endpoint(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", Endpoint, svc.token, method)
}
