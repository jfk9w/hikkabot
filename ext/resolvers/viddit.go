package resolvers

import (
	"context"
	"hikkabot/3rdparty/viddit"

	"github.com/jfk9w-go/flu/httpf"
)

type Viddit viddit.Client

func (r *Viddit) Unmask() *viddit.Client {
	return (*viddit.Client)(r)
}

func (r *Viddit) GetClient(_ httpf.Client) httpf.Client {
	return r.Unmask().Client
}

func (r *Viddit) Resolve(ctx context.Context, _ httpf.Client, url string, _ int64) (string, error) {
	return r.Unmask().ResolveURL(ctx, url)
}
