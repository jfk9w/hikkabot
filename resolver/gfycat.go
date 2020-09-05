package resolver

import (
	"context"
	"net/http"
	"regexp"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

var GfycatRegexp = regexp.MustCompile(`(?i)https://[a-z]+.gfycat.com/[a-z0-9]+?\.mp4`)

type Gfycat struct{}

func (r Gfycat) GetClient() *fluhttp.Client {
	return nil
}

func (r Gfycat) ResolveURL(ctx context.Context, client *fluhttp.Client, url string, _ int64) (string, error) {
	html := flu.NewBuffer()
	if err := client.GET(url).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBodyTo(html).
		Error; err != nil {
		return "", errors.New("get html")
	}

	url = string(GfycatRegexp.Find(html.Bytes()))
	if url == "" {
		return "", errors.New("unable to find URL")
	}

	return url, nil
}

func (r Gfycat) Request(request *fluhttp.Request) *fluhttp.Request {
	return request
}
