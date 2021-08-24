package media

import (
	"context"

	"github.com/pkg/errors"

	fluhttp "github.com/jfk9w-go/flu/http"
)

type PlainResolver struct {
	HttpClient *fluhttp.Client
}

func (r PlainResolver) GetClient(defaultClient *fluhttp.Client) *fluhttp.Client {
	if r.HttpClient != nil {
		return r.HttpClient
	}

	return defaultClient
}

func (r PlainResolver) Resolve(_ context.Context, _ *fluhttp.Client, url string, _ int64) (string, error) {
	return url, nil
}

type ErrorResolver string

func (r ErrorResolver) GetClient(defaultClient *fluhttp.Client) *fluhttp.Client {
	return defaultClient
}

func (r ErrorResolver) Resolve(_ context.Context, _ *fluhttp.Client, _ string, _ int64) (string, error) {
	return "", errors.New(string(r))
}
