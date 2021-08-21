package resolver

import (
	"context"

	"github.com/jfk9w/hikkabot/3rdparty/viddit"

	fluhttp "github.com/jfk9w-go/flu/http"
)

type Viddit struct {
	Client *viddit.Client
}

func (v Viddit) GetClient() *fluhttp.Client {
	return v.Client.HttpClient
}

func (v Viddit) ResolveURL(ctx context.Context, _ *fluhttp.Client, url string, _ int64) (string, error) {
	return v.Client.ResolveURL(ctx, url)
}
