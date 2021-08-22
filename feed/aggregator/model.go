package aggregator

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/feed/executor"
)

type EventListener interface {
	OnResume(ctx context.Context, sub *feed.Subscription) error
	OnSuspend(ctx context.Context, sub *feed.Subscription) error
	OnDelete(ctx context.Context, sub *feed.Subscription) error
	OnClear(ctx context.Context, feedID telegram.ID, pattern string, deleted int64) error
}

type Executor interface {
	Submit(id interface{}, task executor.Task)
}

type HTMLWriterFactory interface {
	CreateHTMLWriter(ctx context.Context, feedID telegram.ID) (*richtext.HTMLWriter, error)
}

type DefaultHTMLWriterFactory struct {
	telegram.Sender
}

func (f DefaultHTMLWriterFactory) CreateHTMLWriter(ctx context.Context, feedID telegram.ID) (*richtext.HTMLWriter, error) {
	return richtext.HTML(ctx, f, true, feedID), nil
}

type Context struct {
	flu.Clock
	feed.Storage
	EventListener
	HTMLWriterFactory

	Interval time.Duration
	Vendors  map[string]feed.Vendor
}
