package feed

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/jfk9w/hikkabot/util/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SuspendListener interface {
	OnSuspend(sub *Subscription, err error)
}

type HTMLWriterFactory interface {
	CreateHTMLWriter(ctx context.Context, feedIDs ...telegram.ID) (*richtext.HTMLWriter, error)
}

type TelegramHTML struct {
	telegram.Sender
}

func (f TelegramHTML) CreateHTMLWriter(ctx context.Context, feedIDs ...telegram.ID) (*richtext.HTMLWriter, error) {
	chatIDs := make([]telegram.ChatID, len(feedIDs))
	for i, feedID := range feedIDs {
		chatIDs[i] = telegram.ID(feedID)
	}

	return richtext.HTML(ctx, telegram.Sender(f), false, chatIDs...), nil
}

type aggregatorTask struct {
	htmlWriterFactory HTMLWriterFactory
	clock             flu.Clock
	store             Storage
	interval          time.Duration
	vendors           map[string]Vendor
	feedID            telegram.ID
	suspendListener   SuspendListener
	metrics           metrics.Registry
}

func (t *aggregatorTask) Execute(ctx context.Context) error {
	for {
		sub, err := t.store.Shift(ctx, t.feedID)
		if err != nil {
			return errors.Wrap(err, "advance")
		}

		if err := t.update(ctx, sub); err != nil {
			updateErr := err
			if ctx.Err() != nil {
				return err
			} else {
				t.metrics.Counter("update_err", sub.MetricsLabels()).Inc()
				if err := t.updateStore(sub.Header, err); err != nil {
					if ctx.Err() != nil {
						return err
					} else {
						logrus.WithFields(sub.Fields()).
							Warnf("update failed: %s", err)
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

func (t *aggregatorTask) update(ctx context.Context, sub *Subscription) error {
	vendor, ok := t.vendors[sub.Vendor]
	if !ok {
		return errors.Errorf("invalid vendor: %s", sub.Vendor)
	}
	queue := NewQueue(sub.Header, sub.Data, 5)
	vctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer close(queue.channel)
		vendor.Refresh(vctx, queue)
	}()
	count := 0
	defer func() { logrus.WithFields(sub.Fields()).Debugf("processed %d updates", count) }()
	for update := range queue.channel {
		if update.Error != nil {
			return errors.Wrap(update.Error, "update")
		}
		html, err := t.htmlWriterFactory.CreateHTMLWriter(ctx, t.feedID)
		if err != nil {
			return errors.Wrap(err, "create HTMLWriter")
		}
		if update.WriteHTML != nil {
			if err := update.WriteHTML(html); err != nil {
				return errors.Wrap(err, "write")
			}
			if err := html.Flush(); err != nil {
				return errors.Wrap(err, "flush")
			}
		}
		err = t.updateStore(sub.Header, update.Data)
		if err != nil {
			return errors.Wrap(err, "store update")
		}
		t.metrics.Counter("update_ok", sub.MetricsLabels()).Inc()
		count++
	}

	if count == 0 {
		err := t.updateStore(sub.Header, sub.Data)
		if err != nil {
			return errors.Wrap(err, "store update")
		}
	}

	return nil
}

var updateStoreTimeout = 10 * time.Second

func (t *aggregatorTask) updateStore(header *Header, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateStoreTimeout)
	defer cancel()
	return t.store.Update(ctx, t.clock.Now(), header, value)
}

type Aggregator struct {
	Clock             flu.Clock
	Executor          TaskExecutor
	Storage           Storage
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
	ids, err := a.Storage.Init(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		a.submitTask(id)
	}

	return nil
}

func (a *Aggregator) submitTask(feedID telegram.ID) {
	a.Executor.Submit(feedID, &aggregatorTask{
		htmlWriterFactory: a.HTMLWriterFactory,
		clock:             a.Clock,
		store:             a.Storage,
		interval:          a.UpdateInterval,
		vendors:           a.Vendors,
		feedID:            feedID,
		suspendListener:   a.SuspendListener,
		metrics:           a.Metrics,
	})
}

func (a *Aggregator) Close() error {
	a.Executor.Close()
	return nil
}

func (a *Aggregator) Subscribe(ctx context.Context, feedID telegram.ID, ref string, options []string) (*Subscription, error) {
	for vendorID, vendor := range a.Vendors {
		sub, err := vendor.Parse(ctx, ref, options)
		switch err {
		case nil:
			data, err := gorm.ToJsonb(sub.Data)
			sub := &Subscription{
				Header: &Header{
					SubID:  sub.SubID,
					Vendor: vendorID,
					FeedID: feedID,
				},
				Name: sub.Name,
				Data: data,
			}

			if err != nil {
				return sub, errors.Wrap(err, "wrap data")
			}

			if err := a.Storage.Create(ctx, sub); err != nil {
				return sub, err
			}

			a.submitTask(feedID)
			return sub, nil

		case ErrWrongVendor:
			continue

		default:
			return nil, err
		}
	}

	return nil, ErrWrongVendor
}

func (a *Aggregator) Suspend(ctx context.Context, header *Header, err error) (*Subscription, error) {
	if err := a.Storage.Update(ctx, a.Clock.Now(), header, err); err != nil {
		return nil, err
	}

	return a.Storage.Get(ctx, header)
}

func (a *Aggregator) Resume(ctx context.Context, header *Header) (*Subscription, error) {
	if err := a.Storage.Update(ctx, a.Clock.Now(), header, nil); err != nil {
		return nil, err
	}

	sub, err := a.Storage.Get(ctx, header)
	if err != nil {
		return nil, err
	}

	a.submitTask(header.FeedID)
	return sub, nil
}

func (a *Aggregator) Delete(ctx context.Context, header *Header) (*Subscription, error) {
	sub, err := a.Storage.Get(ctx, header)
	if err != nil {
		return nil, errors.Wrap(err, "get")
	}

	if err := a.Storage.Delete(ctx, header); err != nil {
		return nil, errors.Wrap(err, "store delete")
	}

	return sub, nil
}

func (a *Aggregator) Clear(ctx context.Context, feedID telegram.ID, pattern string) (int64, error) {
	return a.Storage.DeleteAll(ctx, feedID, pattern)
}

func (a *Aggregator) List(ctx context.Context, feedID telegram.ID, active bool) ([]Subscription, error) {
	return a.Storage.List(ctx, feedID, active)
}
