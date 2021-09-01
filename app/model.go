package app

import (
	"context"
	"io"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	"gorm.io/gorm"

	"github.com/jfk9w/hikkabot/core/event"
	"github.com/jfk9w/hikkabot/core/feed"
	"github.com/jfk9w/hikkabot/core/media"
)

type Interface interface {
	flu.Clock
	GetVersion() string
	GetConfig(value interface{}) error
	GetMediaManager(ctx context.Context) (*media.Manager, error)
	GetDatabase() (*gorm.DB, error)
	GetEventStorage(ctx context.Context) (event.Storage, error)
	Manage(service io.Closer)
}

type VendorPlugin interface {
	VendorID() string
	CreateVendor(ctx context.Context, app Interface) (feed.Vendor, error)
}

type ConverterPlugin interface {
	ConverterID() string
	MIMETypes() []string
	CreateConverter(ctx context.Context, app Interface) (media.Converter, error)
}

type Config struct {
	Aconvert struct {
		Servers []int
		Probe   *aconvert.Probe
	}
}
