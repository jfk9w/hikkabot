package resolver

import (
	"context"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/feed"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

var AconvertMIMETypes = map[string]string{
	"video/webm": "mp4",
}

var AconvertMIMETypeArray = make([]string, len(AconvertMIMETypes))

func init() {
	i := 0
	for mimeType := range AconvertMIMETypes {
		AconvertMIMETypeArray[i] = mimeType
		i++
	}
}

type Aconvert struct {
	*aconvert.Client
}

func (a Aconvert) MIMETypes() []string {
	return AconvertMIMETypeArray
}

func (a Aconvert) Convert(ctx context.Context, ref *feed.MediaRef) (format.MediaRef, error) {
	format := AconvertMIMETypes[ref.MIMEType]
	resp, err := a.Client.Convert(ctx, flu.URL(ref.ResolvedURL), aconvert.Opts{}.TargetFormat(format))
	if err != nil {
		return nil, errors.Wrap(err, "convert")
	}

	return &feed.MediaRef{
		MediaResolver: feed.DummyMediaResolver{Client: a.Client.Client},
		Manager:       ref.Manager,
		URL:           resp.URL(),
		Dedup:         ref.Dedup,
		Blob:          ref.Blob,
		FeedID:        ref.FeedID,
	}, nil
}
