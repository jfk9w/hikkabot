package output

import (
	"context"
)

type Interface interface {
	WriteUnbreakable(ctx context.Context, text string) error
	Flush(ctx context.Context) error
}
