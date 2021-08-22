package executor

import (
	"context"
)

type Task interface {
	Execute(ctx context.Context) error
}
