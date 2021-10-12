package plugin

import (
	"context"

	"hikkabot/app"
	"hikkabot/core/feed"
	. "hikkabot/ext/vendors/dvach/catalog"

	"github.com/pkg/errors"
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
