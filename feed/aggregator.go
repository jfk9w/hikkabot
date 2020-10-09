package feed

import (
	"context"
	"log"
	"time"

	"github.com/jfk9w-go/flu/metrics"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type SuspendListener interface {
	OnSuspend(sub Sub, err error)
}

type HTMLWriterFactory interface {
	CreateHTMLWriter(ctx context.Context, feedIDs ...ID) (*format.HTMLWriter, error)
}

type TelegramHTML struct {
	telegram.Sender
}

func (f TelegramHTML) CreateHTMLWriter(ctx context.Context, feedIDs ...ID) (*format.HTMLWriter, error) {
	chatIDs := make([]telegram.ChatID, len(feedIDs))
	for i, feedID := range feedIDs {
		chatIDs[i] = telegram.ID(feedID)
	}

	return format.HTML(ctx, telegram.Sender(f), false, chatIDs...), nil
}

type aggregatorTask struct {
	htmlWriterFactory HTMLWriterFactory
	store             Feeds
	interval          time.Duration
	vendors           map[string]Vendor
	feedID            ID
	suspendListener   SuspendListener
	metrics           metrics.Registry
}

func (t *aggregatorTask) Execute(ctx context.Context) error {
	for {
		sub, err := t.store.Advance(ctx, t.feedID)
		if err != nil {
			return errors.Wrap(err, "advance")
		}

		if err := t.update(ctx, sub); err != nil {
			updateErr := err
			if ctx.Err() != nil {
				return err
			} else {
				t.metrics.Counter("update_err", sub.MetricsLabels()).Inc()
				if err := t.updateStore(sub.SubID, err); err != nil {
					if ctx.Err() != nil {
						return err
					} else {
						log.Printf("[sub > %s] update failed: %s", sub.SubID, err)
					}
				} else if t.suspendListener != nil {
					go t.suspendListener.OnSuspend(sub, updateErr)
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(t.interval):
			continue
		}
	}
}

func (t *aggregatorTask) update(ctx context.Context, sub Sub) error {
	vendor, ok := t.vendors[sub.Vendor]
	if !ok {
		return errors.Errorf("invalid vendor: %s", sub.Vendor)
	}
	queue := NewQueue(sub.SubID, 5)
	vctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go vendor.Load(vctx, sub.Data, queue)
	count := 0
	defer func() { log.Printf("[sub > %s] processed %d updates", sub.SubID, count) }()
	for update := range queue.channel {
		if update.Error != nil {
			return errors.Wrap(update.Error, "update")
		}
		html, err := t.htmlWriterFactory.CreateHTMLWriter(ctx, t.feedID)
		if err != nil {
			return errors.Wrap(err, "create HTMLWriter")
		}
		if err := update.Write(html); err != nil {
			return errors.Wrap(err, "write")
		}
		if err := html.Flush(); err != nil {
			return errors.Wrap(err, "flush")
		}
		data, err := DataFrom(update.Data)
		if err != nil {
			return errors.Wrap(err, "wrap data")
		}
		err = t.updateStore(sub.SubID, data)
		if err != nil {
			return errors.Wrap(err, "store update")
		}
		t.metrics.Counter("update_ok", sub.MetricsLabels()).Inc()
		count++
	}

	if count == 0 {
		err := t.updateStore(sub.SubID, sub.Data)
		if err != nil {
			return errors.Wrap(err, "store update")
		}
	}

	return nil
}

var updateStoreTimeout = 10 * time.Second

func (t *aggregatorTask) updateStore(subID SubID, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateStoreTimeout)
	defer cancel()
	return t.store.Update(ctx, subID, value)
}

type Aggregator struct {
	Executor          TaskExecutor
	Feeds             Feeds
	HTMLWriterFactory HTMLWriterFactory
	Vendors           map[string]Vendor
	UpdateInterval    time.Duration
	SuspendListener   SuspendListener
	Metrics           metrics.Registry
}

func (a *Aggregator) Vendor(id string, vendor Vendor) *Aggregator {
	if a.Metrics == nil {
		a.Metrics = metrics.DummyRegistry{}
	}

	if a.Vendors == nil {
		a.Vendors = map[string]Vendor{}
	}

	a.Vendors[id] = vendor
	return a
}

func (a *Aggregator) Init(ctx context.Context, suspendListener SuspendListener) error {
	a.SuspendListener = suspendListener
	ids, err := a.Feeds.Init(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		a.submitTask(id)
	}

	return nil
}

func (a *Aggregator) submitTask(feedID ID) {
	a.Executor.Submit(feedID, &aggregatorTask{
		htmlWriterFactory: a.HTMLWriterFactory,
		store:             a.Feeds,
		interval:          a.UpdateInterval,
		vendors:           a.Vendors,
		feedID:            feedID,
		suspendListener:   a.SuspendListener,
		metrics:           a.Metrics,
	})
}

func (a *Aggregator) Close() error {
	a.Executor.Close()
	return a.Feeds.Close()
}

func (a *Aggregator) Subscribe(ctx context.Context, feedID ID, ref string, options []string) (Sub, error) {
	for vendorID, vendor := range a.Vendors {
		sub, err := vendor.Parse(ctx, ref, options)
		switch err {
		case nil:
			data, err := DataFrom(sub.Data)
			sub := Sub{
				SubID: SubID{
					ID:     sub.ID,
					Vendor: vendorID,
					FeedID: feedID,
				},
				Name: sub.Name,
				Data: data,
			}

			if err != nil {
				return sub, errors.Wrap(err, "wrap data")
			}

			if err := a.Feeds.Create(ctx, sub); err != nil {
				return sub, err
			}

			a.submitTask(feedID)
			return sub, nil

		case ErrWrongVendor:
			continue

		default:
			return Sub{}, err
		}
	}

	return Sub{}, ErrWrongVendor
}

func (a *Aggregator) Suspend(ctx context.Context, subID SubID, err error) (Sub, error) {
	if err := a.Feeds.Update(ctx, subID, err); err != nil {
		return Sub{}, err
	}
	return a.Feeds.Get(ctx, subID)
}

func (a *Aggregator) Resume(ctx context.Context, subID SubID) (Sub, error) {
	if err := a.Feeds.Update(ctx, subID, nil); err != nil {
		return Sub{}, err
	}
	sub, err := a.Feeds.Get(ctx, subID)
	if err != nil {
		return Sub{}, err
	}

	a.submitTask(subID.FeedID)
	return sub, nil
}

func (a *Aggregator) Delete(ctx context.Context, subID SubID) (Sub, error) {
	sub, err := a.Feeds.Get(ctx, subID)
	if err != nil {
		return Sub{}, errors.Wrap(err, "get")
	}
	if err := a.Feeds.Delete(ctx, subID); err != nil {
		return Sub{}, errors.Wrap(err, "store delete")
	}
	return sub, nil
}

func (a *Aggregator) Clear(ctx context.Context, feedID ID, pattern string) (int64, error) {
	return a.Feeds.Clear(ctx, feedID, pattern)
}

func (a *Aggregator) List(ctx context.Context, feedID ID, active bool) ([]Sub, error) {
	return a.Feeds.List(ctx, feedID, active)
}
