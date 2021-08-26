package media

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/telegram-bot-api/ext/media"
)

type Manager struct {
	*Context
	flu.RateLimiter
	ctx    context.Context
	work   flu.WaitGroup
	cancel func()
}

func (m *Manager) Init(ctx context.Context) {
	m.ctx, m.cancel = context.WithCancel(ctx)
}

func (m *Manager) Close() error {
	m.cancel()
	m.work.Wait()
	return nil
}

func (m *Manager) Submit(ref *Ref) media.Ref {
	ref.Context = m.Context
	v := media.NewVar()
	m.work.Go(m.ctx, func(ctx context.Context) {
		if err := m.RateLimiter.Start(ctx); err != nil {
			v.Set(nil, err)
			return
		}

		defer m.RateLimiter.Complete()
		v.Set(ref.Get(ctx))
	})

	return v
}
