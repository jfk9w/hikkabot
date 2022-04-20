package feed

import (
	"github.com/pkg/errors"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrExists          = errors.New("exists")
	ErrForbidden       = errors.New("forbidden")
	ErrWrongVendor     = errors.New("wrong vendor")
	ErrSuspendedByUser = errors.New("suspended by user")
	ErrInvalidHeader   = errors.New("invalid header")
)

const Deadborn = "deadborn"
