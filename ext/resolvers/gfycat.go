package resolvers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"hikkabot/feed/media"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/httpf"
)

type GfycatLike[C any] struct {
	Name string
}

func (r GfycatLike[C]) String() string {
	return "media-resolver." + r.Name
}

func (r *GfycatLike[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	return nil
}

func (r *GfycatLike[C]) Resolve(ctx context.Context, source *url.URL) (media.MetaRef, error) {
	if !strings.Contains(source.Host, r.Name) {
		return nil, nil
	}

	url := strings.Trim(source.String(), "/")
	lastSlash := strings.LastIndex(url, "/")
	code := url[lastSlash+1:]

	var resp struct {
		GfyItem struct {
			URL string `json:"mp4Url"`
		} `json:"gfyItem"`
	}

	apiURL := fmt.Sprintf("https://api.%s.com/v1/gfycats/%s", r.Name, code)
	if err := httpf.GET(apiURL).
		Exchange(ctx, nil).
		CheckStatus(http.StatusOK).
		DecodeBody(flu.JSON(&resp)).
		Error(); err != nil {
		return nil, err
	}

	return &media.HTTPRef{
		URL:    resp.GfyItem.URL,
		Buffer: true,
	}, nil
}
