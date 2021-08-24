package app

import (
	"context"

	"github.com/jfk9w/hikkabot/core/media"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w/hikkabot/core/feed"
)

type VendorPlugin interface {
	VendorID() string
	CreateVendor(ctx context.Context, app *Instance) (feed.Vendor, error)
}

type ConverterPlugin interface {
	ConverterID() string
	MIMETypes() []string
	CreateConverter(ctx context.Context, app *Instance) (media.Converter, error)
}

type Config struct {
	Aconvert struct {
		Servers []int
		Probe   *aconvert.Probe
	}
}
