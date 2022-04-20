package media

import (
	"context"
	"mime"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

type HTTPRef struct {
	Client httpf.Client
	URL    string
	Meta   *Meta
	Buffer bool
}

func (r HTTPRef) GetMeta(ctx context.Context) (*Meta, error) {
	if r.Meta != nil {
		return r.Meta, nil
	}

	var m Meta
	return &m, r.exchange(ctx, http.MethodHead).HandleFunc(func(resp *http.Response) error {
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			return errors.New("content type is empty")
		}

		var err error
		m.MIMEType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			return errors.Wrapf(err, "invalid content type: %s", contentType)
		}

		contentLength := resp.Header.Get("Content-Length")
		size, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			m.Size = -1
		} else {
			m.Size = Size(size)
		}

		return nil
	}).Error()
}

func (r HTTPRef) Get(ctx context.Context) (flu.Input, error) {
	if r.Client != nil {
		return r.exchange(ctx, http.MethodGet), nil
	}

	return flu.URL(r.URL), nil
}

func (r HTTPRef) exchange(ctx context.Context, method string) *httpf.ExchangeResult {
	return httpf.Request(r.URL).
		Method(method).
		Exchange(ctx, r.Client).
		CheckStatus(http.StatusOK)
}

type LocalRef struct {
	Input flu.Input
	Meta  *Meta
}

func (r LocalRef) GetMeta(ctx context.Context) (*Meta, error) {
	return r.Meta, nil
}

func (r LocalRef) Get(ctx context.Context) (flu.Input, error) {
	return r.Input, nil
}

type LazyRef struct {
	Ref
	Meta *Meta
}

func (r LazyRef) GetMeta(ctx context.Context) (*Meta, error) {
	return r.Meta, nil
}
