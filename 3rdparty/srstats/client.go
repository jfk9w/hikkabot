package srstats

import (
	"context"
	"net/http"

	"github.com/jfk9w-go/flu/logf"
	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu/apfel"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
)

var BaseURL = "https://subredditstats.com/api"

type Client[C any] struct {
	client httpf.Client
}

func (c Client[C]) String() string {
	return "srstats.client"
}

func (c *Client[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	c.client = new(http.Client)
	return nil
}

func (c *Client[C]) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	logf.Get(c).Resultf(req.Context(), logf.Trace, logf.Warn, "%s => %v", &httpf.RequestBuilder{Request: req}, err)
	return resp, err
}

func (c *Client[C]) GetGlobalHistogram(ctx context.Context) (map[string]float64, error) {
	var m map[string]float64
	return m, httpf.GET(BaseURL+"/globalSubredditsIdHist").
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&m)).
		Error()
}

func (c *Client[C]) GetHistogram(ctx context.Context, subreddit string) (map[string]float64, error) {
	var buf flu.ByteBuffer
	if err := httpf.GET(BaseURL+"/subredditNameToSubredditsHist").
		Query("subredditName", subreddit).
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		CopyBody(&buf).
		Error(); err != nil {
		return nil, err
	}

	var resp map[string]float64
	if err := flu.DecodeFrom(&buf, flu.JSON(&resp)); err == nil {
		return resp, nil
	}

	return nil, errors.Errorf(buf.String())
}

func (c *Client[C]) GetSubredditNames(ctx context.Context, ids []string) ([]string, error) {
	names := make([]string, 0)
	return names, httpf.POST(BaseURL+"/specificSubredditIdsToNames", flu.JSON(map[string][]string{"subredditIds": ids})).
		ContentType("text/plain").
		Exchange(ctx, c).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&names)).
		Error()
}
