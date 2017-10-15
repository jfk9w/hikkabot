package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	Result      *json.RawMessage    `json:"result"`
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
	return json.Unmarshal(r.Result, v)
}

type DeferredRequest struct {
	Method     string
	Parameters url.Values
	Callback   func(resp *Response, err error)
}

const (
	GatewayAlreadyRunning = "gateway already running"
	GatewayNotRunning     = "gateway not running"
	GatewayError          = "gateway error"
)

type Gateway struct {
	client *http.Client
	token  string
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
		queue:  make(chan DeferredRequest, 100000),
		mu:     &sync.Mutex{},
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

	g.wg = &sync.WaitGroup{}
	g.wg.Add(1)
	go func() {
		retries := 0
		ticker := time.NewTicker(60 * time.Millisecond)
		defer func() {
			ticker.Stop()
			g.wg.Done()
		}()

		for {
			select {
			case <-g.choke:
				return

			case r := <-g.queue:
				<-ticker.C
				var (
					resp *Response
					err  error
				)

				for {
					resp, err = g.MakeRequest(r.Method, r.Parameters)
					if err == nil {
						retries = 0
						if resp.Parameters != nil {
							timeout := resp.Parameters.RetryAfter
							if timeout > 0 {
								time.Sleep(timeout)
								select {
								case <-g.choke:
									return
								default:
									continue
								}
							}
						}

						break
					}

					if retries >= 30 {
						break
					}

					time.Sleep(retries * time.Second)
					select {
					case <-g.choke:
						return
					default:
						continue
					}
				}

				if r.Callback {
					r.Callback(resp, err)
				}

			case <-g.stop:
				return
			}
		}
	}()

	return nil
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

func (g *Gateway) MakeRequest(method string, parameters url.Values) (*Response, error) {
	r, err := http.PostForm(g.endpoint(method), parameters)
	if err != nil {
		return nil, error
	}

	defer r.Body.Close()

	resp := new(Response)
	err := json.NewDecoder(r.Body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (g *Gateway) Submit(request DeferredRequest) {
	g.queue <- request
}

func (g *Gateway) endpoint(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", Endpoint, g.token, method)
}
