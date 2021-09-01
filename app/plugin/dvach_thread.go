package plugin

import (
	"context"

	"github.com/pkg/errors"

	"github.com/jfk9w/hikkabot/app"
	"github.com/jfk9w/hikkabot/core/feed"
	. "github.com/jfk9w/hikkabot/ext/vendors/dvach/thread"
)

type DvachThread DvachClient

func (p *DvachThread) Unmask() *DvachClient {
	return (*DvachClient)(p)
}

func (p *DvachThread) VendorID() string {
	return "2ch/thread"
}

func (p *DvachThread) CreateVendor(ctx context.Context, app app.Interface) (feed.Vendor, error) {
	client, err := p.Unmask().Get(app)
	if client == nil {
		return nil, errors.Wrap(err, "create dvach client")
	}

	mediaManager, err := app.GetMediaManager(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get media manager")
	}

	return &Vendor{
		DvachClient:  client,
		MediaManager: mediaManager,
		GetTimeout:   p.getTimeout,
	}, err
}
