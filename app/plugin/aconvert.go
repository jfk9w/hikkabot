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
	*Config `yaml:"-,inline"`
}

type Aconvert []string

func (p Aconvert) ConverterID() string {
	return "aconvert"
}

func (p Aconvert) MIMETypes() []string {
	return p
}

func (p Aconvert) CreateConverter(ctx context.Context, app app.Interface) (media.Converter, error) {
	globalConfig := new(struct {
		Aconvert AconvertConfig
	})

	if err := app.GetConfig().As(globalConfig); err != nil {
		return nil, errors.Wrap(err, "get config")
	}

	config := globalConfig.Aconvert
	if !config.Enabled {
		return nil, nil
	}

	metrics, err := app.GetMetricsRegistry(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get metrics registry")
	}

	client := NewClient(ctx, metrics.WithPrefix("aconvert"), config.Config)
	return (*converters.Aconvert)(client), nil
}
