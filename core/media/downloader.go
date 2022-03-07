package media

import (
	"context"
	"net/http"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/backoff"
	httpf "github.com/jfk9w-go/flu/httpf"
)

type downloader struct {
	httpf.Client
	retries int
}

func (d *downloader) DownloadMetadata(ctx context.Context, url string) (*Metadata, error) {
	metadata := new(Metadata)
	return metadata, backoff.Retry{
		Retries: d.retries,
		Backoff: backoff.Const(time.Second),
		Body: func(ctx context.Context) error {
			return httpf.HEAD(url).
				Exchange(ctx, d).
				CheckStatus(http.StatusOK).
				Handle(metadata).
				Error()
		},
	}.Do(ctx)
}

func (d *downloader) DownloadContents(ctx context.Context, url string, out flu.Output) error {
	return backoff.Retry{
		Retries: d.retries,
		Backoff: backoff.Const(time.Second),
		Body: func(ctx context.Context) error {
			return httpf.GET(url).
				Exchange(ctx, d).
				CheckStatus(http.StatusOK).
				DecodeBodyTo(out).
				Error()
		},
	}.Do(ctx)
}
