package converters

import (
	"context"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
	"github.com/pkg/errors"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	. "hikkabot/core/media"
)

var FFmpegMIMETypes = map[string][4]string{
	"video/webm": {"mp4", "libx264", "aac", "video/mp4"},
}

var FFmpeg Converter = _ffmpeg{}

type _ffmpeg struct{}

func (_ffmpeg) ID() string {
	return "ffmpeg"
}

func (_ffmpeg) Convert(_ context.Context, ref *Ref) (media.Ref, error) {
	if target, ok := FFmpegMIMETypes[ref.MIMEType]; !ok {
		return nil, errors.Errorf("unsupported mime type: %s", ref.MIMEType)
	} else if blob, err := ref.Alloc(ref.Now()); err != nil {
		return nil, errors.Wrap(err, "allocate blob")
	} else if file, ok := blob.(flu.File); !ok {
		return nil, errors.Wrapf(err, "only flu.File is supported, got %T", blob)
	} else if err := ffmpeg.Input(ref.ResolvedURL).
		Output(file.Path(), ffmpeg.KwArgs{"c": "copy", "f": target[0], "c:v": target[1], "c:a": target[2]}).
		Run(); err != nil {
		return nil, errors.Wrap(err, "copy")
	} else {
		return &FFmpegResult{
			MIMEType: target[3],
			Input:    file,
		}, nil
	}
}

type FFmpegResult media.Value

func (r *FFmpegResult) Get(_ context.Context) (*media.Value, error) {
	return (*media.Value)(r), nil
}
