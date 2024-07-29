package aconvert

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/backoff"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
)

// Probe denotes a file to be used for discovering servers.
type Probe struct {
	File flu.File `yaml:"file" doc:"Path to the file which will be used for testing (discovering) servers."`
	// Format is the target conversion format for the File.
	Format string `yaml:"format" doc:"Target conversion format for the file." example:"mp4"`
}

type Config struct {
	ServerIDs  []int        `yaml:"serverIds,omitempty" doc:"Server IDs to use for conversion." default:"[3, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29]"`
	Probe      *Probe       `yaml:"probe,omitempty" doc:"Probe parameters for checking servers. If set, servers from serverIds list will be tested before adding to the client pool."`
	Timeout    flu.Duration `yaml:"timeout,omitempty" doc:"Timeout to use while making HTTP requests." default:"5m"`
	MaxRetries int          `yaml:"maxRetries,omitempty" doc:"Max request retries before giving up." default:"3"`
}

// Context is the application configuration interface.
type Context interface {
	AconvertConfig() Config
}

// Client is a mixin encapsulating aconvert.com client.
type Client[C Context] struct {
	*client
}

func (c *Client[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	return c.Standalone(ctx, app.Config().AconvertConfig())
}

// Standalone allows to initialize the Client outside github.com/jfk9w-go/flu/apfel application context.
// It is recommended to create Config instance with apfel.Default[Config]() in this case to properly initialize
// default value.
func (c *Client[C]) Standalone(ctx context.Context, config Config) error {
	transport := httpf.NewDefaultTransport()
	transport.ResponseHeaderTimeout = config.Timeout.Value
	client := &client{
		client:     &http.Client{Transport: withReferer(transport)},
		servers:    make(chan server, len(config.ServerIDs)),
		maxRetries: config.MaxRetries,
	}

	if config.Probe == nil {
		for _, id := range config.ServerIDs {
			client.servers <- client.makeServer(ctx, id)
		}
	} else {
		_, _ = syncf.Go(ctx, func(ctx context.Context) {
			client.discover(ctx, config.Probe, config.ServerIDs)
		})
	}

	c.client = client
	configData, _ := flu.ToString(flu.PipeInput(apfel.JSONViaYAML(config)))
	logf.Get(c).Tracef(ctx, "started with %s", configData)
	return nil
}

type client struct {
	client     httpf.Client
	servers    chan server
	maxRetries int
}

func (c *client) String() string {
	return "aconvert.client"
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	logf.Get(c).Resultf(req.Context(), logf.Trace, logf.Warn, "%s => %v", httpf.RequestBuilder{Request: req}, err)
	return resp, err
}

// Convert converts the provided media and returns a response.
func (c *client) Convert(ctx context.Context, in flu.Input, opts Options) (*Response, error) {
	var resp *Response
	retry := backoff.Retry{
		Retries: c.maxRetries,
		Backoff: backoff.Const(0),
		Body: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case server := <-c.servers:
				defer func() {
					if ctx.Err() == nil {
						select {
						case <-ctx.Done():
						case c.servers <- server:
						}
					}
				}()

				var err error
				resp, err = c.convert(ctx, server, in, opts)
				return err
			}
		},
	}

	return resp, retry.Do(ctx)
}

func (c *client) convert(ctx context.Context, server server, in flu.Input, opts Options) (*Response, error) {
	req, err := opts.Code(81000).makeRequest(server.convertURL, in)
	if err != nil {
		return nil, errors.Wrap(err, "make request")
	}

	var resp Response
	if err := req.Exchange(ctx, c).DecodeBody(&resp).Error(); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *client) discover(ctx context.Context, probe *Probe, serverIDs []int) {
	var discovered int32
	var work syncf.WaitGroup
	for i := range serverIDs {
		serverID := serverIDs[i]
		_, _ = syncf.GoWith(ctx, work.Spawn, func(ctx context.Context) {
			server := c.makeServer(ctx, serverID)
			retry := backoff.Retry{
				Retries: c.maxRetries,
				Backoff: backoff.Exp{Base: time.Second, Factor: 1.5},
				Body: func(ctx context.Context) error {
					_, err := c.convert(ctx, server, probe.File, make(Options).TargetFormat(probe.Format))
					return err
				},
			}

			err := retry.Do(ctx)
			logf.Get(c).Resultf(ctx, logf.Debug, logf.Warn, "server %d init: %v", err)
			if err == nil {
				atomic.AddInt32(&discovered, 1)
				c.servers <- server
			}
		})
	}

	work.Wait()
	if discovered == 0 {
		logf.Get(c).Errorf(ctx, "no hosts discovered")
	} else {
		logf.Get(c).Infof(ctx, "discovered %d servers", discovered)
	}
}

func (c *client) makeServer(ctx context.Context, id interface{}) server {
	value := host(id) + "/convert/convert4.php"
	if _, err := url.Parse(value); err != nil {
		logf.Get(c).Panicf(ctx, "invalid convert-batch URL: %s", err)
	}

	return server{value, id}
}

func host(serverID interface{}) string {
	return fmt.Sprintf("https://s%v.aconvert.com", serverID)
}

type server struct {
	convertURL string
	id         interface{}
}

func withReferer(rt http.RoundTripper) httpf.RoundTripperFunc {
	return func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Referer", "https://www.aconvert.com/")
		req.Header.Set("Origin", "https://www.aconvert.com")
		return rt.RoundTrip(req)
	}
}
