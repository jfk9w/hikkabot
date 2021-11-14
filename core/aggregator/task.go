package aggregator

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"
	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v3"

	"hikkabot/core/feed"
)

type Task struct {
	*Context
	me3x.Registry
	FeedID telegram.ID
}

func (t *Task) Execute(ctx context.Context) error {
	for {
		sub, err := t.Storage.Shift(ctx, t.FeedID)
		if err != nil {
			return err
		}

		log := logrus.WithFields(sub.Labels().Map())
		if updates, err := t.refresh(ctx, sub); err != nil {
			if flu.IsContextRelated(err) {
				return err
			}

			sub.Error = null.StringFrom(err.Error())
			if err := t.Storage.Update(ctx, t.Now(), sub.Header, err); err != nil {
				if flu.IsContextRelated(err) {
					return err
				}
				log.Warnf("update in storage: %s", err)
			} else {
				log.Debugf("suspended: %s", sub.Error.String)
				if err := t.OnSuspend(ctx, t.Client, sub); err != nil {
					return err
				}
			}
		} else {
			log.Debugf("refresh: %d updates ok", updates)
		}

		if err := flu.Sleep(ctx, t.Interval); err != nil {
			return err
		}
	}
}

func (t *Task) refresh(ctx context.Context, sub *feed.Subscription) (int, error) {
	vendor, ok := t.Vendors[sub.Vendor]
	if !ok {
		return 0, feed.ErrWrongVendor
	}

	queue := feed.NewQueue(sub.Header, sub.Data, t.Preload)
	defer new(flu.WaitGroup).Go(ctx, func(ctx context.Context) {
		defer close(queue.C)
		vendor.Refresh(ctx, queue)
	})()

	count := 0
	for update := range queue.C {
		if err := update.Error; err != nil {
			return 0, err
		}

		if update.WriteHTML != nil {
			html := t.createHTMLWriter(ctx)
			if err := update.WriteHTML(html); err != nil {
				return 0, errors.Wrap(err, "write HTML")
			}

			if err := html.Flush(); err != nil {
				return 0, errors.Wrap(err, "flush HTML")
			}
		}

		if err := t.Update(ctx, t.Now(), sub.Header, update.Data); err != nil {
			return 0, errors.Wrap(err, "update in storage")
		}

		t.Counter("update", sub.Labels()).Inc()
		logrus.WithFields(sub.Labels().Map()).Debug("update: ok")
		count++
	}

	if count == 0 {
		if err := t.Update(ctx, t.Now(), sub.Header, sub.Data); err != nil {
			return 0, errors.Wrap(err, "update in storage")
		}
	}

	return count, nil
}

func (t *Task) createHTMLWriter(ctx context.Context) *tghtml.Writer {
	return &tghtml.Writer{
		Context: ctx,
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    t.Client,
				ID:        t.FeedID,
				Silent:    true,
				ParseMode: telegram.HTML,
			},
			PageSize: tghtml.DefaultMaxMessageSize * 9 / 10,
		},
	}
}
