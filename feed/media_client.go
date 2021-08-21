package feed

import (
	"context"
	"mime"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

const (
	ContentTypeHeader   = "Content-Type"
	ContentLengthHeader = "Content-Length"
)

type MediaMetadata struct {
	Size     int64
	MIMEType string
}

func (m *MediaMetadata) Handle(resp *http.Response) error {
	return m.Fill(resp.Header.Get(ContentTypeHeader), resp.Header.Get(ContentLengthHeader))
}

func (m *MediaMetadata) Fill(contentType, contentLength string) error {
	var err error
	m.MIMEType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return errors.Wrapf(err, "invalid %s: %s", ContentTypeHeader, contentType)
	}

	m.Size, err = strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		m.Size = UnknownSize
	}

	return nil
}

type MediaClient interface {
	Metadata(ctx context.Context, url string) (*MediaMetadata, error)
	Contents(ctx context.Context, url string, out flu.Output) error
}

type DefaultMediaClient struct {
	HttpClient *fluhttp.Client
	Retries    int
}

func (c *DefaultMediaClient) Metadata(ctx context.Context, url string) (*MediaMetadata, error) {
	m := new(MediaMetadata)
	return m, c.HttpClient.HEAD(url).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		HandleResponse(m).
		Error
}

func (c *DefaultMediaClient) Contents(ctx context.Context, url string, out flu.Output) error {
	return c.HttpClient.GET(url).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		DecodeBodyTo(out).
		Error
}
