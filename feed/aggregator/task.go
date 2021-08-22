package aggregator

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	null "gopkg.in/guregu/null.v3"
)

type Task struct {
	*Context
	metrics.Registry
	FeedID telegram.ID
}

func (t *Task) Execute(ctx context.Context) error {
	for {
		sub, err := t.Storage.Shift(ctx, t.FeedID)
		if err != nil {
			return err
		}

		log := logrus.WithFields(sub.Fields())
		if err := t.refresh(ctx, sub); err != nil {
			if isContextRelated(err) {
				return err
			}

			sub.Error = null.StringFrom(err.Error())
			if err := t.Storage.Update(ctx, t.Now(), sub.Header, err); err != nil {
				if isContextRelated(err) {
					return err
				}

				logrus.WithFields(sub.Fields()).Warnf("update in storage: %s", err)
			} else {
				log.Debugf("suspended: %s", sub.Error.String)
				if err := t.OnSuspend(ctx, sub); err != nil {
					return err
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(t.Interval):
			continue
		}
	}
}

func (t *Task) refresh(ctx context.Context, sub *feed.Subscription) error {
	vendor, ok := t.Vendors[sub.Vendor]
	if !ok {
		return feed.ErrWrongVendor
	}

	queue := feed.NewQueue(sub.Header, sub.Data, 5)
	defer new(flu.WaitGroup).Go(ctx, func(ctx context.Context) {
		defer close(queue.C)
		vendor.Refresh(ctx, queue)
	})()

	hasUpdates := false
	for update := range queue.C {
		if err := update.Error; err != nil {
			return err
		}

		html, err := t.CreateHTMLWriter(ctx, t.FeedID)
		if err != nil {
			return errors.Wrap(err, "create HTML writer")
		}

		if update.WriteHTML != nil {
			if err := update.WriteHTML(html); err != nil {
				return errors.Wrap(err, "write HTML")
			}

			if err := html.Flush(); err != nil {
				return errors.Wrap(err, "flush HTML")
			}
		}

		if err := t.Update(ctx, t.Now(), sub.Header, update.Data); err != nil {
			return errors.Wrap(err, "update in storage")
		}

		t.Counter("update", metrics.Labels(sub.Fields())).Inc()
		hasUpdates = true
	}

	if !hasUpdates {
		if err := t.Update(ctx, t.Now(), sub.Header, sub.Data); err != nil {
			return errors.Wrap(err, "update in storage")
		}
	}

	return nil
}

func isContextRelated(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
