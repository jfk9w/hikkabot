package resolvers

import (
	"context"

	"hikkabot/3rdparty/viddit"

	fluhttp "github.com/jfk9w-go/flu/http"
)

type Viddit viddit.Client

func (r *Viddit) Unmask() *viddit.Client {
	return (*viddit.Client)(r)
}

func (r *Viddit) GetClient(_ *fluhttp.Client) *fluhttp.Client {
	return r.Unmask().HttpClient
}

func (r *Viddit) Resolve(ctx context.Context, _ *fluhttp.Client, url string, _ int64) (string, error) {
	return r.Unmask().ResolveURL(ctx, url)
}
