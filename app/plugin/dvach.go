package plugin

import (
	"time"

	"github.com/jfk9w-go/flu"
	httpf "github.com/jfk9w-go/flu/httpf"

	"hikkabot/3rdparty/dvach"
	"hikkabot/app"
)

type DvachConfig struct {
	Enabled    bool
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

	globalConfig := new(struct{ Dvach DvachConfig })
	if err := app.GetConfig().As(globalConfig); err != nil {
		return nil, err
	}

	config := globalConfig.Dvach
	if !config.Enabled {
		return nil, nil
	}

	c.value = dvach.NewClient(httpf.NewClient(nil), config.Usercode)
	c.getTimeout = config.GetTimeout.GetOrDefault(30 * time.Second)
	return c.value, nil
}
