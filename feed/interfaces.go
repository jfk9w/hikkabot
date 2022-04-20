package feed

import (
	"context"
	"time"

	"hikkabot/feed/media"

	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

type Refresh interface {
	Init(ctx context.Context, data any) error
	Submit(ctx context.Context, writeHTML WriteHTML, data any) error
}

type Vendor interface {
	String() string
	Parse(ctx context.Context, ref string, options []string) (*Draft, error)
	Refresh(ctx context.Context, header Header, queue Refresh) error
}

type BeforeResumeListener interface {
	Vendor
	BeforeResume(ctx context.Context, header Header) error
}

type AfterStateListener interface {
	AfterResume(ctx context.Context, sub *Subscription) error
	AfterSuspend(ctx context.Context, sub *Subscription) error
	AfterDelete(ctx context.Context, sub *Subscription) error
	AfterClear(ctx context.Context, feedID ID, pattern string, deleted int64) error
}

type Poller interface {
	Subscribe(ctx context.Context, feedID ID, ref string, options []string) error
	Suspend(ctx context.Context, header Header, err error) error
	Resume(ctx context.Context, header Header) error
	Delete(ctx context.Context, header Header) error
	Clear(ctx context.Context, feedID ID, pattern string) error
}

type Blobs interface {
	Buffer(mimeType string, ref media.Ref) media.MetaRef
}

type EventTx interface {
	GetLastEventData(feedID ID, eventType string, filter map[string]any, value any) error
	SaveEvent(feedID ID, eventType string, value any) error
	DeleteEvents(feedID ID, types []string, filter map[string]any) error
	CountEventsByType(feedID ID, types []string, filter map[string]any) (map[string]int64, error)
}

type EventStorage interface {
	EventTx(ctx context.Context, tx func(tx EventTx) error) error
	SaveEvent(ctx context.Context, feedID ID, eventType string, value any) error
	CountEventsBy(ctx context.Context, feedID ID, since time.Time, key string, multipliers map[string]float64) (map[string]int64, error)
}

type MediaHashStorage interface {
	IsMediaUnique(ctx context.Context, hash *MediaHash) (bool, error)
}

type Tx interface {
	GetSubscription(id Header) (*Subscription, error)
	DeleteSubscription(header Header) error
	UpdateSubscription(header Header, value any) error
}

type Storage interface {
	Tx(ctx context.Context, tx func(tx Tx) error) error
	GetActiveFeedIDs(ctx context.Context) ([]ID, error)
	GetSubscription(ctx context.Context, id Header) (*Subscription, error)
	CreateSubscription(ctx context.Context, sub *Subscription) error
	ShiftSubscription(ctx context.Context, feedID ID) (*Subscription, error)
	ListSubscriptions(ctx context.Context, feedID ID, active bool) ([]Subscription, error)
	DeleteAllSubscriptions(ctx context.Context, feedID ID, pattern string) (int64, error)
	UpdateSubscription(ctx context.Context, header Header, value any) error
}

type TaskExecutor interface {
	Submit(id any, task Task)
}

type Mediator interface {
	Mediate(ctx context.Context, url string, dedupKey *ID) receiver.MediaRef
}
