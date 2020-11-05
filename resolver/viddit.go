package resolver

import (
	"context"

	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w/hikkabot/vendors/common"
)

type Viddit struct {
	Client *common.Viddit
}

func (v Viddit) GetClient() *fluhttp.Client {
	return v.Client.Client
}

func (v Viddit) ResolveURL(ctx context.Context, _ *fluhttp.Client, url string, _ int64) (string, error) {
	return v.Client.Get(ctx, url)
}
