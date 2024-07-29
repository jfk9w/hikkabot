package apfel

import (
	"context"
	"fmt"
)

// MixinAny allows to apply dependency management to a value based on its type.
type MixinAny[C any, V any] struct {
	Value V
}

func (m MixinAny[C, V]) String() string {
	return fmt.Sprintf("any.%T", m.Value)
}

func (m MixinAny[C, V]) Include(ctx context.Context, app MixinApp[C]) error {
	return nil
}
