package aggregator

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/core/executor"
	"github.com/jfk9w/hikkabot/core/feed"
)

type EventListener interface {
	OnResume(ctx context.Context, client telegram.Client, sub *feed.Subscription) error
	OnSuspend(ctx context.Context, client telegram.Client, sub *feed.Subscription) error
	OnDelete(ctx context.Context, client telegram.Client, sub *feed.Subscription) error
	OnClear(ctx context.Context, client telegram.Client, feedID telegram.ID, pattern string, deleted int64) error
}

type Executor interface {
	Submit(id interface{}, task executor.Task)
}

type Context struct {
	flu.Clock
	feed.Storage
	EventListener
	telegram.Client

	Interval time.Duration
	Preload  int
	Vendors  map[string]feed.Vendor
}
