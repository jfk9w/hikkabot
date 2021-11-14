package resolvers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jfk9w-go/flu"

	httpf "github.com/jfk9w-go/flu/httpf"
)

type Gfycat string

func (r Gfycat) GetClient(defaultClient *httpf.Client) *httpf.Client {
	return defaultClient
}

func (r Gfycat) Resolve(ctx context.Context, client *httpf.Client, url string, _ int64) (string, error) {
	url = strings.Trim(url, "/")
	lastSlash := strings.LastIndex(url, "/")
	code := url[lastSlash+1:]

	resp := new(struct {
		GfyItem struct {
			URL string `json:"mp4Url"`
		} `json:"gfyItem"`
	})

	apiURL := fmt.Sprintf("https://api.%s.com/v1/gfycats/%s", string(r), code)
	if err := client.GET(apiURL).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(resp)).
		Error; err != nil {
		return "", err
	}

	return resp.GfyItem.URL, nil
}
