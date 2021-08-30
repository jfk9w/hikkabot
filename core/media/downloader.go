package media

import (
	"context"
	"net/http"
	"time"

	"github.com/jfk9w-go/flu/backoff"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
)

type downloader struct {
	*fluhttp.Client
	retries int
}

func (d *downloader) DownloadMetadata(ctx context.Context, url string) (*Metadata, error) {
	metadata := new(Metadata)
	return metadata, backoff.Retry{
		Retries: d.retries,
		Backoff: backoff.Const(time.Second),
		Body: func(ctx context.Context) error {
			return d.HEAD(url).
				Context(ctx).
				Execute().
				CheckStatus(http.StatusOK).
				HandleResponse(metadata).
				Error
		},
	}.Do(ctx)
}

func (d *downloader) DownloadContents(ctx context.Context, url string, out flu.Output) error {
	return backoff.Retry{
		Retries: d.retries,
		Backoff: backoff.Const(time.Second),
		Body: func(ctx context.Context) error {
			return d.GET(url).
				Context(ctx).
				Execute().
				CheckStatus(http.StatusOK).
				DecodeBodyTo(out).
				Error
		},
	}.Do(ctx)
}
