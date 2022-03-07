package aggregator

import (
	"context"

	"gopkg.in/guregu/null.v3"

	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"

	"hikkabot/core/feed"
)

type OnResumeListener interface {
	OnResume(ctx context.Context, data gormf.JSONB) error
}

type Default struct {
	*Context
	me3x.Registry
	Executor
}

func (a *Default) Init(ctx context.Context) error {
	activeFeedIDs, err := a.Active(ctx)
	if err != nil {
		return errors.Wrap(err, "get active feeds from storage")
	}

	for _, feedID := range activeFeedIDs {
		a.submitTask(feedID)
	}

	return nil
}

const Deadborn = "deadborn"

func (a *Default) Subscribe(ctx context.Context, feedID telegram.ID, ref string, options []string) error {
	for vendorKey, vendor := range a.Vendors {
		draft, err := vendor.Parse(ctx, ref, options)
		if err == feed.ErrWrongVendor {
			continue
		} else if err != nil {
			return errors.Wrap(err, "parse")
		}

		data, err := gormf.ToJSONB(draft.Data)
		if err != nil {
			return errors.Wrap(err, "convert data")
		}

		sub := &feed.Subscription{
			Header: &feed.Header{
				SubID:  draft.SubID,
				Vendor: vendorKey,
				FeedID: feedID,
			},
			Name: draft.Name,
			Data: data,
		}

		for _, option := range options {
			if option == Deadborn {
				sub.Error = null.StringFrom(Deadborn)
			}
		}

		if err := a.Create(ctx, sub); err != nil {
			return errors.Wrap(err, "create in storage")
		}

		if sub.Error.IsZero() {
			if listener, ok := vendor.(OnResumeListener); ok {
				if err := listener.OnResume(ctx, sub.Data); err != nil {
					return errors.Wrap(err, "call on resume listener")
				}
			}

			a.submitTask(sub.FeedID)
			if err := a.OnResume(ctx, a.Client, sub); err != nil {
				return err
			}
		}

		return nil
	}

	return feed.ErrWrongVendor
}

func (a *Default) Suspend(ctx context.Context, header *feed.Header, err error) error {
	if err := a.Update(ctx, a.Now(), header, err); err != nil {
		return err
	}

	sub, err := a.Get(ctx, header)
	if err != nil {
		return errors.Wrap(err, "get from storage")
	}

	return a.OnSuspend(ctx, a.Client, sub)
}

func (a *Default) Resume(ctx context.Context, header *feed.Header) error {
	if err := a.Update(ctx, a.Now(), header, nil); err != nil {
		return err
	}

	sub, err := a.Get(ctx, header)
	if err != nil {
		return errors.Wrap(err, "get from storage")
	}

	vendor, ok := a.Vendors[sub.Vendor]
	if !ok {
		return errors.Errorf("no vendor for %s", sub.Vendor)
	}

	if listener, ok := vendor.(OnResumeListener); ok {
		if err := listener.OnResume(ctx, sub.Data); err != nil {
			return errors.Wrap(err, "call on resume listener")
		}
	}

	a.submitTask(header.FeedID)
	return a.OnResume(ctx, a.Client, sub)
}

func (a *Default) Delete(ctx context.Context, header *feed.Header) error {
	sub, err := a.Get(ctx, header)
	if err != nil {
		return errors.Wrap(err, "get from storage")
	}

	if err := a.Storage.Delete(ctx, header); err != nil {
		return err
	}

	return a.OnDelete(ctx, a.Client, sub)
}

func (a *Default) Clear(ctx context.Context, feedID telegram.ID, pattern string) error {
	deleted, err := a.DeleteAll(ctx, feedID, pattern)
	if err != nil {
		return err
	}

	return a.OnClear(ctx, a.Client, feedID, pattern, deleted)
}

func (a *Default) submitTask(feedID telegram.ID) {
	a.Submit(feedID, &Task{
		Context:  a.Context,
		Registry: a.WithPrefix("refresh"),
		FeedID:   feedID,
	})
}
