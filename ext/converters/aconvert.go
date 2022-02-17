package converters

import (
	"context"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
	. "hikkabot/core/media"
)

var AconvertMIMETypes = map[string]string{
	"video/webm": "mp4",
}

type Aconvert aconvert.Client

func (c *Aconvert) Unmask() *aconvert.Client {
	return (*aconvert.Client)(c)
}

func (c *Aconvert) ID() string {
	return "aconvert"
}

func (c *Aconvert) Convert(ctx context.Context, ref *Ref) (media.Ref, error) {
	format := AconvertMIMETypes[ref.MIMEType]
	resp, err := c.Unmask().Convert(ctx, flu.URL(ref.ResolvedURL), aconvert.Opts{}.TargetFormat(format))
	if err != nil {
		return nil, errors.Wrap(err, "convert")
	}

	return &Ref{
		Resolver: PlainResolver{HttpClient: c.Unmask().Client},
		Context:  ref.Context,
		URL:      resp.URL(),
		Dedup:    ref.Dedup,
		Blob:     ref.Blob,
		FeedID:   ref.FeedID,
	}, nil
}
