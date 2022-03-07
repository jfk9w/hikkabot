package media

import (
	"context"

	"github.com/jfk9w-go/flu/httpf"

	"github.com/pkg/errors"
)

type PlainResolver struct {
	httpf.Client
}

func (r PlainResolver) GetClient(defaultClient httpf.Client) httpf.Client {
	if r.Client != nil {
		return r.Client
	}

	return defaultClient
}

func (r PlainResolver) Resolve(_ context.Context, _ httpf.Client, url string, _ int64) (string, error) {
	return url, nil
}

type ErrorResolver string

func (r ErrorResolver) GetClient(defaultClient httpf.Client) httpf.Client {
	return defaultClient
}

func (r ErrorResolver) Resolve(_ context.Context, _ httpf.Client, _ string, _ int64) (string, error) {
	return "", errors.New(string(r))
}
