package feed

import (
	"github.com/pkg/errors"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrSuspendedByUser = errors.New("suspended by user")
	ErrUnsupported     = errors.New("unsupported")
)

const Deadborn = "deadborn"
