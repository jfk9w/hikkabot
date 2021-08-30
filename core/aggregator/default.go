package aggregator

import (
	"context"

	gormutil "github.com/jfk9w-go/flu/gorm"

	"github.com/jfk9w/hikkabot/core/feed"

	"github.com/jfk9w-go/flu/metrics"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Default struct {
	*Context
	metrics.Registry
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

func (a *Default) Subscribe(ctx context.Context, feedID telegram.ID, ref string, options []string) error {
	for vendorKey, vendor := range a.Vendors {
		draft, err := vendor.Parse(ctx, ref, options)
		if err == feed.ErrWrongVendor {
			continue
		} else if err != nil {
			return errors.Wrap(err, "parse")
		}

		data, err := gormutil.ToJSONB(draft.Data)
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

		if err := a.Create(ctx, sub); err != nil {
			return errors.Wrap(err, "create in storage")
		}

		a.submitTask(sub.FeedID)
		if err := a.OnResume(ctx, a.Client, sub); err != nil {
			return err
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
