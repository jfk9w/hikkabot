package feed

import (
	"context"
	"time"

	"hikkabot/feed/media"

	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

// Refresh is an interface for submitting updates from subscription vendors.
type Refresh interface {
	// Init is called in order to read Vendor specific data from subscription.
	Init(ctx context.Context, data any) error
	// Submit submits a subscription update.
	// If an error is returned no further updates may be submitted.
	Submit(ctx context.Context, writeHTML WriteHTML, data any) error
}

// Vendor is a subscription vendor.
// It is responsible for parsing user input parameters and collecting subscription updates.
type Vendor interface {
	// String is Vendor identifier.
	// It is used for distinguishing subscriptions for different vendors, and thus must be unique.
	String() string
	// Parse is called for parsing user input parameters.
	// If this Vendor does not support provided input, (nil, nil)-tuple should be returned.
	Parse(ctx context.Context, ref string, options []string) (*Draft, error)
	// Refresh collects subscription updates and submits them to Refresh interface.
	Refresh(ctx context.Context, header Header, refresh Refresh) error
}

// BeforeResumeListener interface may be implemented by a Vendor in order to handle "before resume" events.
type BeforeResumeListener interface {
	Vendor
	// BeforeResume is called before subscription creation or resuming.
	// If an error is returned, the subscription will not be created (or resumed).
	BeforeResume(ctx context.Context, header Header) error
}

// AfterStateListener handles subscription state changes.
type AfterStateListener interface {
	AfterResume(ctx context.Context, sub *Subscription) error
	AfterSuspend(ctx context.Context, sub *Subscription) error
	AfterDelete(ctx context.Context, sub *Subscription) error
	AfterClear(ctx context.Context, feedID ID, pattern string, deleted int64) error
}

// Poller is the core application interface which provides means to manage subscriptions.
type Poller interface {
	// Subscribe creates a subscription from user input if it does not exist yet.
	Subscribe(ctx context.Context, feedID ID, ref string, options []string) error
	// Suspend suspends a previously created or resumed subscription.
	Suspend(ctx context.Context, header Header, err error) error
	// Resume resumes a previously suspended subscription.
	Resume(ctx context.Context, header Header) error
	// Delete deletes a subscription.
	Delete(ctx context.Context, header Header) error
	// Clear deletes all subscriptions whose error message matches pattern.
	Clear(ctx context.Context, feedID ID, pattern string) error
}

// Blobs provides means for temporary large memory allocation for media downloads.
type Blobs interface {
	Buffer(mimeType string, ref media.Ref) media.MetaRef
}

// EventTx represents a database transaction on "event data subset".
type EventTx interface {
	// GetLastEventData collects data into `value` from last Event matching `filter` and `eventType`.
	GetLastEventData(feedID ID, eventType string, filter map[string]any, value any) error
	// SaveEvent creates and saves new Event.
	SaveEvent(feedID ID, eventType string, value any) error
	// DeleteEvents deletes all events matching `types` and `filter`.
	DeleteEvents(feedID ID, types []string, filter map[string]any) error
	// CountEventsByType returns total number of events matching `filter` aggregated by `types`.
	CountEventsByType(feedID ID, types []string, filter map[string]any) (map[string]int64, error)
}

// EventStorage provides access to "event data subset" in storage.
type EventStorage interface {
	// EventTx executes a transaction.
	EventTx(ctx context.Context, tx func(tx EventTx) error) error
	// SaveEvent creates and saves new Event.
	SaveEvent(ctx context.Context, feedID ID, eventType string, value any) error
	// CountEventsBy returns an aggregated statistic by event types since some moment in time.
	// Aggregation is done by event data `key`, each event type receives weight from `multipliers` map.
	CountEventsBy(ctx context.Context, feedID ID, since time.Time, key string, multipliers map[string]float64) (map[string]int64, error)
}

// MediaHashStorage keeps track of duplicate media.
type MediaHashStorage interface {
	// IsMediaUnique returns `true` if passed `hash` is not present in storage yet.
	IsMediaUnique(ctx context.Context, hash *MediaHash) (bool, error)
}

// Tx represents a database transaction on "subscription data subset".
type Tx interface {
	// GetSubscription is an "alias" for Storage.GetSubscription.
	GetSubscription(header Header) (*Subscription, error)
	// DeleteSubscription deletes a Subscription.
	DeleteSubscription(header Header) error
	// UpdateSubscription is an "alias" for Storage.UpdateSubscription.
	UpdateSubscription(header Header, value any) error
}

type Storage interface {
	// Tx executes a transaction.
	Tx(ctx context.Context, tx func(tx Tx) error) error
	// GetActiveFeedIDs returns a slice of active feed IDs.
	GetActiveFeedIDs(ctx context.Context) ([]ID, error)
	// GetSubscription returns a Subscription.
	GetSubscription(ctx context.Context, id Header) (*Subscription, error)
	// CreateSubscription creates new Subscription.
	CreateSubscription(ctx context.Context, sub *Subscription) error
	// ShiftSubscription returns "next" active Subscription if any.
	ShiftSubscription(ctx context.Context, feedID ID) (*Subscription, error)
	// ListSubscriptions lists all active or suspended subscriptions.
	ListSubscriptions(ctx context.Context, feedID ID, active bool) ([]Subscription, error)
	// DeleteAllSubscriptions deletes all subscriptions with error message matching `pattern`.
	DeleteAllSubscriptions(ctx context.Context, feedID ID, pattern string) (int64, error)
	// UpdateSubscription updates Subscription data and error.
	// `value` can be either:
	//   nil – this sets Subscription error to nil (applicable only to suspended subscriptions)
	//   non-nil error – this sets Subscription error (applicable only to active subscriptions)
	//   gormf.JSONB – this updates the Subscription data (applicable only to active subscriptions)
	UpdateSubscription(ctx context.Context, header Header, value any) error
}

// TaskExecutor is responsible for running background subscription update tasks.
type TaskExecutor interface {
	// Submit submits a Task for execution if no Task with the same `id` is being executed already.
	Submit(id any, task Task)
}

// Mediator is responsible for downloading and converting media files.
type Mediator interface {
	Mediate(ctx context.Context, url string, dedupKey *ID) receiver.MediaRef
}
