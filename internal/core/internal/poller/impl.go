package poller

import (
	"context"
	"time"

	"github.com/jfk9w/hikkabot/v4/internal/feed"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"
)

type Impl struct {
	Clock    syncf.Clock
	Storage  feed.Storage
	Executor feed.TaskExecutor
	Metrics  me3x.Registry
	Telegram telegram.Client
	Interval time.Duration
	Preload  int

	vendors        map[string]feed.Vendor
	stateListeners StateListeners
}

func (p *Impl) String() string {
	return ServiceID
}

func (p *Impl) RegisterVendor(id string, vendor feed.Vendor) error {
	if p.vendors == nil {
		p.vendors = make(map[string]feed.Vendor)
	}

	if _, ok := p.vendors[id]; ok {
		return errors.Errorf("vendor %s already registered", id)
	}

	p.vendors[id] = vendor
	return nil
}

func (p *Impl) RegisterStateListener(listener feed.AfterStateListener) {
	p.stateListeners = append(p.stateListeners, listener)
}

func (p *Impl) RestoreActive(ctx context.Context) error {
	activeFeedIDs, err := p.Storage.GetActiveFeedIDs(ctx)
	if err != nil {
		return errors.Wrap(err, "get active feeds from storage")
	}

	for _, feedID := range activeFeedIDs {
		p.submitTask(feedID)
	}

	return nil
}

func (p *Impl) Subscribe(ctx context.Context, feedID feed.ID, ref string, options []string) error {
	for vendorKey, vendor := range p.vendors {
		draft, err := vendor.Parse(ctx, ref, options)
		switch {
		case err != nil:
			return errors.Wrapf(err, "parse with %s", vendorKey)
		case draft == nil:
			continue
		}

		header := feed.Header{
			SubID:  draft.SubID,
			Vendor: vendorKey,
			FeedID: feedID,
		}

		if listener, ok := vendor.(feed.BeforeResumeListener); ok {
			err := listener.BeforeResume(ctx, header)
			logf.Get(p).Resultf(ctx, logf.Debug, logf.Warn, "before resume [%s]: %v", header, err)
			if err != nil {
				return err
			}
		}

		data, err := gormf.ToJSONB(draft.Data)
		if err != nil {
			return errors.Wrap(err, "convert data")
		}

		sub := &feed.Subscription{
			Header: header,
			Name:   draft.Name,
			Data:   data,
		}

		for _, option := range options {
			if option == feed.Deadborn {
				sub.Error = null.StringFrom(feed.Deadborn)
			}
		}

		if err := p.Storage.CreateSubscription(ctx, sub); err != nil {
			return errors.Wrap(err, "create in storage")
		}

		if sub.Error.IsZero() {
			p.submitTask(sub.FeedID)
			p.stateListeners.OnResume(ctx, sub)
			return nil
		}

		p.stateListeners.OnSuspend(ctx, sub)
		return nil
	}

	return errors.New("failed to find matching vendor")
}

func (p *Impl) Suspend(ctx context.Context, header feed.Header, err error) error {
	var sub *feed.Subscription
	if err := p.Storage.Tx(ctx, func(tx feed.Tx) error {
		if err := tx.UpdateSubscription(header, err); err != nil {
			return err
		}

		sub, err = tx.GetSubscription(header)
		return err
	}); err != nil {
		return err
	}

	p.stateListeners.OnSuspend(ctx, sub)
	return nil
}

func (p *Impl) Resume(ctx context.Context, header feed.Header) error {
	vendor, ok := p.vendors[header.Vendor]
	if !ok {
		return errors.Errorf("no vendor for %s", header.Vendor)
	}

	if listener, ok := vendor.(feed.BeforeResumeListener); ok {
		err := listener.BeforeResume(ctx, header)
		logf.Get(p).Resultf(ctx, logf.Debug, logf.Warn, "before resume [%s]: %v", header, err)
		if err != nil {
			return err
		}
	}

	var sub *feed.Subscription
	if err := p.Storage.Tx(ctx, func(tx feed.Tx) error {
		if err := tx.UpdateSubscription(header, nil); err != nil {
			return err
		}

		var err error
		sub, err = tx.GetSubscription(header)
		return err
	}); err != nil {
		return err
	}

	p.submitTask(header.FeedID)
	p.stateListeners.OnResume(ctx, sub)
	return nil
}

func (p *Impl) Delete(ctx context.Context, header feed.Header) error {
	var sub *feed.Subscription
	if err := p.Storage.Tx(ctx, func(tx feed.Tx) error {
		var err error
		sub, err = tx.GetSubscription(header)
		if err != nil {
			return err
		}

		return tx.DeleteSubscription(header)
	}); err != nil {
		return err
	}

	p.stateListeners.OnDelete(ctx, sub)
	return nil
}

func (p *Impl) Clear(ctx context.Context, feedID feed.ID, pattern string) error {
	deleted, err := p.Storage.DeleteAllSubscriptions(ctx, feedID, pattern)
	if err != nil {
		return err
	}

	p.stateListeners.OnDeleteAll(ctx, feedID, pattern, deleted)
	return nil
}

func (p *Impl) submitTask(feedID feed.ID) {
	p.Executor.Submit(feedID, func(ctx context.Context) error {
		for {
			sub, err := p.Storage.ShiftSubscription(ctx, feedID)
			if err != nil {
				return err
			}

			updates, err := p.refresh(ctx, sub)
			logf.Get(p).Resultf(ctx, logf.Debug, logf.Warn, "received %d updates for [%s]: %v", updates, sub, err)
			switch {
			case syncf.IsContextRelated(err):
				return err
			case err != nil:
				sub.Error = null.StringFrom(err.Error())
				err := p.Storage.UpdateSubscription(ctx, sub.Header, err)
				logf.Get(p).Resultf(ctx, logf.Trace, logf.Warn, "update [%s] in db: %v", sub, err)
				switch {
				case syncf.IsContextRelated(err):
					return err
				case err == nil:
					p.stateListeners.OnSuspend(ctx, sub)
				}
			}

			if err := flu.Sleep(ctx, p.Interval); err != nil {
				return err
			}
		}
	})
}

func (p *Impl) refresh(ctx context.Context, sub *feed.Subscription) (int, error) {
	vendor, ok := p.vendors[sub.Vendor]
	if !ok {
		return 0, errors.Errorf("vendor [%s] not found", sub.Vendor)
	}

	header := sub.Header
	queue := newUpdateQueue(sub.Data, p.Preload)
	defer syncf.GoSync(ctx, func(ctx context.Context) {
		defer queue.close()
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = errors.Errorf("%+v", r)
				}

				_ = queue.cancel(ctx, err)
			}
		}()

		logf.Get(p).Debugf(ctx, "starting [%s] refresh now", sub)
		if err := vendor.Refresh(ctx, header, queue); err != nil {
			_ = queue.cancel(ctx, err)
		}
	})()

	count := 0
	for update := range queue.channel {
		if err := update.err; err != nil {
			return 0, err
		}

		if update.writeHTML != nil {
			html := p.createHTMLWriter(ctx, sub.FeedID)
			if err := update.writeHTML(html); err != nil {
				return 0, errors.Wrap(err, "write HTML")
			}

			if err := html.Flush(); err != nil {
				return 0, errors.Wrap(err, "flush HTML")
			}
		}

		if err := p.Storage.UpdateSubscription(ctx, header, update.data); err != nil {
			return 0, errors.Wrap(err, "update in storage")
		}

		logf.Get(p).Tracef(ctx, "update [%s]: ok", sub)
		p.Metrics.Counter("refresh_update", header.Labels()).Inc()
		count++
	}

	if count == 0 {
		if err := p.Storage.UpdateSubscription(ctx, header, sub.Data); err != nil {
			return 0, errors.Wrap(err, "update in storage")
		}
	}

	return count, nil
}

func (p *Impl) createHTMLWriter(ctx context.Context, feedID feed.ID) *tghtml.Writer {
	return (&tghtml.Writer{
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    p.Telegram,
				ID:        telegram.ID(feedID),
				Silent:    true,
				ParseMode: telegram.HTML,
			},
		},
	}).WithContext(output.With(ctx, tghtml.DefaultMaxMessageSize*9/10, 0))
}
