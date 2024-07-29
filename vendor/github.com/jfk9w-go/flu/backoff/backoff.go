package backoff

import (
	"math"
	"time"

	"golang.org/x/exp/rand"
)

// Const represents a fixed duration backoff strategy.
type Const time.Duration

func (b Const) Timeout(int) time.Duration {
	return time.Duration(b)
}

// Exp provides exponential backoff `(Base * Factor ^ (retry - 1))`.
type Exp struct {
	Base   time.Duration
	Factor float64
}

func (b Exp) Timeout(retry int) time.Duration {
	return time.Duration(float64(b.Base) * math.Pow(float64(b.Factor), float64(retry-1)))
}

// Rand implements backoff with random timeouts in [Min, Max) interval.
type Rand struct {
	Min, Max time.Duration
}

func (b Rand) Timeout(int) time.Duration {
	return time.Duration(rand.Float64()*float64(b.Max-b.Min)) + b.Min
}
