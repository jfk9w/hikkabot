package plugin

import (
	"context"

	"github.com/pkg/errors"

	"github.com/jfk9w/hikkabot/app"
	"github.com/jfk9w/hikkabot/core/feed"
	. "github.com/jfk9w/hikkabot/ext/vendors/dvach/catalog"
)

type DvachCatalog DvachClient

func (p *DvachCatalog) Unmask() *DvachClient {
	return (*DvachClient)(p)
}

func (p *DvachCatalog) VendorID() string {
	return "2ch/catalog"
}

func (p *DvachCatalog) CreateVendor(ctx context.Context, app app.Interface) (feed.Vendor, error) {
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
