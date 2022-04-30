package converters

import (
	"context"
	"os"
	"os/exec"

	"hikkabot/core"
	"hikkabot/feed"
	"hikkabot/feed/media"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/pkg/errors"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const ffmpegServiceID = "media-converters.ffmpeg"

type ffmpegFormat struct {
	f, vc, ac string
}

var ffmpegFormats = map[string]ffmpegFormat{
	"video/mp4": {"mp4", "libx264", "aac"},
}

type FFmpeg[C core.BlobContext] struct {
	clock syncf.Clock
	blobs feed.Blobs
}

func (c FFmpeg[C]) String() string {
	return ffmpegServiceID
}

func (c *FFmpeg[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	_, err := exec.LookPath("ffmpeg")
	logf.Get(c).Resultf(ctx, logf.Info, logf.Warn, "check ffmpeg in $PATH: %v", err)
	if err != nil {
		return apfel.ErrDisabled
	}

	var blobs core.Blobs[C]
	if err := app.Use(ctx, &blobs, false); err != nil {
		return err
	}

	c.clock = app
	c.blobs = &blobs
	return nil
}

func (c *FFmpeg[C]) Convert(ctx context.Context, ref media.Ref, mimeType string) (media.MetaRef, error) {
	format, ok := ffmpegFormats[mimeType]
	if !ok {
		return nil, nil
	}

	input, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var stream *ffmpeg.Stream
	switch input := input.(type) {
	case flu.File:
		stream = ffmpeg.Input(input.String())
	case flu.URL:
		stream = ffmpeg.Input(input.String())
	default:
		input, err := c.blobs.Buffer("", syncf.Val[flu.Input]{V: input}).Get(ctx)
		if err != nil {
			return nil, err
		}

		file, ok := input.(flu.File)
		if !ok {
			return nil, errors.Errorf("only flu.File blobs are supported, got %T", input)
		}

		stream = ffmpeg.Input(file.String())
	}

	blob := c.blobs.Buffer(mimeType, syncf.Val[flu.Input]{V: make(flu.Bytes, 0)})
	ctx = core.SkipSizeCheck(ctx)

	output, err := blob.Get(ctx)
	if err != nil {
		return nil, err
	}

	file, ok := output.(flu.File)
	if !ok {
		return nil, errors.Errorf("only flu.File blobs are supported, got %T", output)
	}

	startTime := c.clock.Now()
	stream = stream.Output(file.String(), ffmpeg.KwArgs{
		"c":   "copy",
		"f":   format.f,
		"c:v": format.vc,
		"c:a": format.ac,
	})

	stream.Context = ctx
	err = stream.OverWriteOutput().Run()
	logf.Get(c).Resultf(ctx, logf.Debug, logf.Warn,
		"convert [%s] => [%s] in %s: %v",
		flu.Readable(input), output, c.clock.Now().Sub(startTime), err)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(file.String())
	if err != nil {
		return nil, err
	}

	return &media.LocalRef{
		Input: file,
		Meta: &media.Meta{
			MIMEType: mimeType,
			Size:     media.Size(stat.Size()),
		},
	}, nil
}
