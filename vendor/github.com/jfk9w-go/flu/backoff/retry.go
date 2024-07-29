package backoff

import (
	"context"

	"github.com/jfk9w-go/flu/syncf"

	"github.com/jfk9w-go/flu"
)

// Retry describes a retry strategy.
type Retry struct {

	// Retries is maximum retries.
	Retries int

	// Backoff is a backoff strategy.
	Backoff Interface

	// Body is an action which will be "retry-managed".
	Body func(context.Context) error
}

// Do retries the action until it succeeds or maximum retries is reached.
func (r Retry) Do(ctx context.Context) (err error) {
	err = r.Body(ctx)
	if err == nil || syncf.IsContextRelated(err) {
		return
	}

	for i := 1; r.Retries < 0 || i <= r.Retries; i++ {
		if err = flu.Sleep(ctx, r.Backoff.Timeout(i)); err != nil {
			return
		}

		err = r.Body(ctx)
		if err == nil || syncf.IsContextRelated(err) {
			return
		}
	}

	return
}
