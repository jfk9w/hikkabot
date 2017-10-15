package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
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
	real    Request
	handler ResponseHandler
}

const (
	GatewayAlreadyRunning = "gateway already running"
	GatewayNotRunning     = "gateway not running"
	GatewayChoking        = "gateway choking"
)

type Gateway struct {
	client *http.Client
	token  string
	urgent chan DeferredRequest
	queue  chan DeferredRequest
	wg     *sync.WaitGroup
	mu     *sync.Mutex
	stop   chan struct{}
	choke  chan struct{}
}

func NewGateway(client *http.Client, token string) *Gateway {
	if client == nil {
		client = &http.Client{}
	}

	return &Gateway{
		client: client,
		token:  token,
		urgent: make(chan DeferredRequest, 20),
		queue:  make(chan DeferredRequest, 10000),
		mu:     new(sync.Mutex),
		stop:   make(chan struct{}, 1),
		choke:  make(chan struct{}, 1),
	}
}

func (g *Gateway) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.wg != nil {
		return errors.New(GatewayAlreadyRunning)
	}

	g.wg = new(sync.WaitGroup)
	g.wg.Add(1)
	go func() {
		ticker := time.NewTicker(60 * time.Millisecond)
		defer func() {
			ticker.Stop()
			g.wg.Done()
		}()

		for range ticker.C {
			select {
			case <-g.choke:
				return

			case r := <-g.urgent:
				resp, err := g.RetryRequest(r.real, 2)
				if err.Error() == GatewayChoking {
					return
				}

				if r.handler != nil {
					r.handler(resp, err)
				}

			case r := <-g.queue:
				resp, err := g.RetryRequest(r.real, 5)
				if err.Error() == GatewayChoking {
					return
				}

				if r.handler != nil {
					r.handler(resp, err)
				}

			case <-g.stop:
				return
			}
		}
	}()

	return nil
}

func (g *Gateway) RetryRequest(req Request, retries int) (*Response, error) {
	var (
		resp *Response
		err  error
	)

	for {
		resp, err = g.MakeRequest(req)
		if err == nil {
			if resp.Parameters != nil {
				timeout := time.Duration(resp.Parameters.RetryAfter)
				if timeout > 0 {
					time.Sleep(timeout * time.Second)
					select {
					case <-g.choke:
						return nil, errors.New(GatewayChoking)
					default:
						continue
					}
				}
			}

			break
		}

		if retries == 0 {
			break
		}

		retries--
		time.Sleep(time.Second)
		select {
		case <-g.choke:
			return nil, errors.New(GatewayChoking)
		default:
			continue
		}
	}

	return resp, err
}

func (g *Gateway) Stop(choke bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.wg == nil {
		return errors.New(GatewayNotRunning)
	}

	var c chan struct{}
	if choke {
		c = g.choke
	} else {
		c = g.stop
	}

	c <- unit
	g.wg.Wait()
	g.wg = nil

	return nil
}

func (g *Gateway) MakeRequest(req Request) (*Response, error) {
	r, err := http.PostForm(g.endpoint(req.Method()), req.Parameters())
	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	resp := new(Response)
	err = json.NewDecoder(r.Body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (g *Gateway) Submit(request Request, handler ResponseHandler) {
	g.queue <- DeferredRequest{
		request,
		handler,
	}
}

func (g *Gateway) Urgent(request Request, handler ResponseHandler) {
	g.urgent <- DeferredRequest{
		request,
		handler,
	}
}

func (g *Gateway) endpoint(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", Endpoint, g.token, method)
}
