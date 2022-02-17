package plugin

import (
	"context"

	"hikkabot/app"
	"hikkabot/core/media"
	"hikkabot/ext/converters"

	. "github.com/jfk9w-go/aconvert-api"
	"github.com/pkg/errors"
)

type AconvertConfig struct {
	Enabled bool
	Config  `yaml:"-,inline"`
}

var Aconvert app.ConverterPlugin = aconvert{}

type aconvert struct{}

func (aconvert) ConverterID() string {
	return "aconvert"
}

func (aconvert) MIMETypes() []string {
	values := make([]string, 0, len(converters.AconvertMIMETypes))
	for value := range converters.AconvertMIMETypes {
		values = append(values, value)
	}

	return values
}

func (aconvert) CreateConverter(ctx context.Context, app app.Interface) (media.Converter, error) {
	var globalConfig struct {
		Aconvert AconvertConfig
	}

	if err := app.GetConfig().As(&globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	} else if config := globalConfig.Aconvert; !config.Enabled {
		return nil, nil
	} else if metrics, err := app.GetMetricsRegistry(ctx); err != nil {
		return nil, errors.Wrap(err, "get metrics registry")
	} else {
		client := NewClient(ctx, metrics.WithPrefix("aconvert"), &config.Config)
		return (*converters.Aconvert)(client), nil
	}
}
