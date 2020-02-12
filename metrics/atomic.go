package metrics

import (
	"math"
	"sync/atomic"
)

type AtomicFloat64 uint64

func (f *AtomicFloat64) Add(delta float64) {
	for {
		oldBits := atomic.LoadUint64((*uint64)(f))
		newBits := math.Float64bits(math.Float64frombits(oldBits) + delta)
		if atomic.CompareAndSwapUint64((*uint64)(f), oldBits, newBits) {
			return
		}
	}
}

func (f *AtomicFloat64) Set(value float64) {
	atomic.StoreUint64((*uint64)(f), math.Float64bits(value))
}

func (f *AtomicFloat64) Get() float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(f)))
}

func (f *AtomicFloat64) Swap(value float64) float64 {
	return math.Float64frombits(atomic.SwapUint64((*uint64)(f), math.Float64bits(value)))
}
