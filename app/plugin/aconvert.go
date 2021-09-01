package plugin

import (
	"context"

	"github.com/jfk9w/hikkabot/ext/converters"

	. "github.com/jfk9w-go/aconvert-api"
	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w/hikkabot/app"
	"github.com/jfk9w/hikkabot/core/media"
)

type Aconvert []string

func (p Aconvert) ConverterID() string {
	return "aconvert"
}

func (p Aconvert) MIMETypes() []string {
	return p
}

func (p Aconvert) CreateConverter(ctx context.Context, app app.Interface) (media.Converter, error) {
	globalConfig := new(struct {
		Aconvert struct {
			Servers []int
			Probe   *Probe
		}
	})

	if err := app.GetConfig(globalConfig); err != nil {
		return nil, err
	}

	config := globalConfig.Aconvert
	client := NewClient(fluhttp.NewClient(nil), config.Servers, config.Probe)
	return (*converters.Aconvert)(client), nil
}
