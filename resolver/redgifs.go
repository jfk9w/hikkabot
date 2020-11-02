package resolver

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jfk9w-go/flu"

	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

var redGIFsURLTemplate = "https://api.%s.com/v1/gfycats/%s"

type RedGIFs struct {
	Site string // either redgifs or gfycat
}

func (r RedGIFs) GetClient() *fluhttp.Client {
	return nil
}

func (r RedGIFs) ResolveURL(ctx context.Context, client *fluhttp.Client, url string, _ int64) (string, error) {
	url = strings.Trim(url, "/")
	lastSlash := strings.LastIndex(url, "/")
	code := url[lastSlash+1:]

	resp := new(struct {
		GfyItem struct {
			MP4URL string `json:"mp4Url"`
		} `json:"gfyItem"`
	})

	apiURL := fmt.Sprintf(redGIFsURLTemplate, r.Site, code)
	if err := client.GET(apiURL).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON{Value: resp}).
		Error; err != nil {
		return "", errors.Wrapf(err, "get: %s", apiURL)
	}

	return resp.GfyItem.MP4URL, nil
}

func (r RedGIFs) Request(request *fluhttp.Request) *fluhttp.Request {
	return request
}
