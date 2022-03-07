package plugin

import (
	"time"

	"github.com/pkg/errors"

	"hikkabot/3rdparty/dvach"
	"hikkabot/app"

	"github.com/jfk9w-go/flu"
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

	client, err := dvach.NewClient(nil, config.Usercode)
	if err != nil {
		return nil, errors.Wrap(err, "create dvach client")
	}

	c.value = client
	c.getTimeout = config.GetTimeout.GetOrDefault(30 * time.Second)
	return c.value, nil
}
