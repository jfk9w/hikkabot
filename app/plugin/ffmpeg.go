package plugin

import (
	"context"
	"github.com/pkg/errors"
	"hikkabot/app"
	"hikkabot/core/media"
	"hikkabot/ext/converters"
)

type FFmpegConfig struct {
	Enabled bool
}

var FFmpeg app.ConverterPlugin = ffmpeg{}

type ffmpeg struct{}

func (ffmpeg) ConverterID() string {
	return "ffmpeg"
}

func (ffmpeg) MIMETypes() []string {
	values := make([]string, 0, len(converters.FFmpegMIMETypes))
	for value := range converters.FFmpegMIMETypes {
		values = append(values, value)
	}

	return values
}

func (ffmpeg) CreateConverter(_ context.Context, app app.Interface) (media.Converter, error) {
	var globalConfig struct {
		FFmpeg FFmpegConfig
	}

	if err := app.GetConfig().As(&globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	} else if config := globalConfig.FFmpeg; !config.Enabled {
		return nil, nil
	} else {
		return converters.FFmpeg, nil
	}
}
