package plugin

import (
	"time"

	"github.com/jfk9w-go/flu"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w/hikkabot/3rdparty/dvach"
	"github.com/jfk9w/hikkabot/app"
)

type DvachConfig struct {
	Usercode   string
	GetTimeout flu.Duration
}

type DvachClient struct {
	value      *dvach.Client
	getTimeout time.Duration
}

func (c *DvachClient) Get(app app.Interface) (*dvach.Client, error) {
	if c.value != nil {
		return c.value, nil
	}

	globalConfig := new(struct{ Dvach *DvachConfig })
	if err := app.GetConfig(globalConfig); err != nil {
		return nil, err
	}

	config := globalConfig.Dvach
	if config == nil {
		return nil, nil
	}

	c.value = dvach.NewClient(fluhttp.NewClient(nil), config.Usercode)
	c.getTimeout = config.GetTimeout.GetOrDefault(30 * time.Second)
	return c.value, nil
}
