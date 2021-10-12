package app

import (
	"context"
	"io"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/telegram-bot-api"
	"gorm.io/gorm"

	"hikkabot/core/event"
	"hikkabot/core/feed"
	"hikkabot/core/media"
)

type Interface interface {
	flu.Clock
	GetVersion() string
	GetConfig(value interface{}) error
	GetMetricsRegistry(ctx context.Context) (metrics.Registry, error)
	GetMediaManager(ctx context.Context) (*media.Manager, error)
	GetDefaultDatabase() (*gorm.DB, error)
	GetEventStorage(ctx context.Context) (event.Storage, error)
	GetBot(ctx context.Context) (*telegram.Bot, error)
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
