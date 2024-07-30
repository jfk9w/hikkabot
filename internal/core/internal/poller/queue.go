package poller

import (
	"context"

	"github.com/jfk9w/hikkabot/v4/internal/feed"

	"github.com/jfk9w-go/flu/gormf"
)

type update struct {
	writeHTML feed.WriteHTML
	data      gormf.JSONB
	err       error
}

type updateQueue struct {
	init    gormf.JSONB
	channel chan update
}

func newUpdateQueue(init gormf.JSONB, size int) updateQueue {
	return updateQueue{
		init:    init,
		channel: make(chan update, size),
	}
}

func (q updateQueue) Init(ctx context.Context, value interface{}) error {
	if err := q.init.As(value); err != nil {
		_ = q.cancel(ctx, err)
		return err
	}

	return nil
}

func (q updateQueue) Submit(ctx context.Context, writeHTML feed.WriteHTML, value interface{}) error {
	data, err := gormf.ToJSONB(value)
	if err != nil {
		return err
	}

	update := update{
		writeHTML: writeHTML,
		data:      data,
	}

	return q.submit(ctx, update)
}

func (q updateQueue) cancel(ctx context.Context, err error) error {
	return q.submit(ctx, update{err: err})
}

func (q updateQueue) submit(ctx context.Context, update update) error {
	select {
	case q.channel <- update:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q updateQueue) close() {
	close(q.channel)
}
