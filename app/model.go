package app

import (
	"context"
	"io"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"gorm.io/gorm"

	"github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/core/event"
	"github.com/jfk9w/hikkabot/core/feed"
	"github.com/jfk9w/hikkabot/core/media"
)

type Interface interface {
	flu.Clock
	GetVersion() string
	GetConfig(value interface{}) error
	GetMetricsRegistry(ctx context.Context) (metrics.Registry, error)
	GetMediaManager(ctx context.Context) (*media.Manager, error)
	GetDatabase() (*gorm.DB, error)
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

type OptionalFloat64 float64

func (v *OptionalFloat64) GetOrDefault(defaultValue float64) float64 {
	if v == nil {
		return defaultValue
	}

	return float64(*v)
}

type OptionalInt64 int64

func (v *OptionalInt64) GetOrDefault(defaultValue int64) int64 {
	if v == nil {
		return defaultValue
	}

	return int64(*v)
}
