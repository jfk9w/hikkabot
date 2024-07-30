package converters

import (
	"context"

	"github.com/jfk9w/hikkabot/v4/internal/feed/media"

	"github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
)

var aconvertFormats = map[string]string{
	"video/mp4": "mp4",
}

type Aconvert[C aconvert.Context] struct {
	*aconvert.Client[C]
}

func (c Aconvert[C]) String() string {
	return "media-converters.aconvert"
}

func (c *Aconvert[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var client aconvert.Client[C]
	if err := app.Use(ctx, &client, false); err != nil {
		return err
	}

	c.Client = &client
	return nil
}

func (c *Aconvert[C]) Convert(ctx context.Context, ref media.Ref, mimeType string) (media.MetaRef, error) {
	format, ok := aconvertFormats[mimeType]
	if !ok {
		return nil, nil
	}

	input, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Convert(ctx, input, aconvert.Options{}.TargetFormat(format))
	logf.Get(c).Resultf(ctx, logf.Debug, logf.Warn, "convert %s: %v", flu.Readable(input), err)
	if err != nil {
		return nil, err
	}

	return &media.HTTPRef{
		URL:    resp.URL(),
		Buffer: true,
	}, nil
}
