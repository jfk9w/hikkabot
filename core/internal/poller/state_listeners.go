package poller

import (
	"context"

	"hikkabot/feed"

	"github.com/jfk9w-go/flu/logf"
)

type StateListeners []feed.AfterStateListener

func (ls StateListeners) OnResume(ctx context.Context, sub *feed.Subscription) {
	for _, l := range ls {
		err := l.AfterResume(ctx, sub)
		logf.Get(l).Resultf(ctx, logf.Trace, logf.Warn, "on resume [%s]: %v", sub, err)
	}
}

func (ls StateListeners) OnSuspend(ctx context.Context, sub *feed.Subscription) {
	for _, l := range ls {
		err := l.AfterSuspend(ctx, sub)
		logf.Get(l).Resultf(ctx, logf.Trace, logf.Warn, "on suspend [%s]: %v", sub, err)
	}
}

func (ls StateListeners) OnDelete(ctx context.Context, sub *feed.Subscription) {
	for _, l := range ls {
		err := l.AfterDelete(ctx, sub)
		logf.Get(l).Resultf(ctx, logf.Trace, logf.Warn, "on delete [%s]: %v", sub, err)
	}
}

func (ls StateListeners) OnDeleteAll(ctx context.Context, feedID feed.ID, pattern string, deleted int64) {
	for _, l := range ls {
		err := l.AfterClear(ctx, feedID, pattern, deleted)
		logf.Get(l).Resultf(ctx, logf.Trace, logf.Warn, "on delete all [%d, '%s', %d]: %v", feedID, pattern, deleted, err)
	}
}
