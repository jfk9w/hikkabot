package me3x

import (
	"math"
	"sync/atomic"
)

// AtomicFloat64 is an atomic float64 implementation.
type AtomicFloat64 uint64

// Add adds delta to the value.
func (f *AtomicFloat64) Add(delta float64) {
	for {
		oldBits := atomic.LoadUint64((*uint64)(f))
		newBits := math.Float64bits(math.Float64frombits(oldBits) + delta)
		if atomic.CompareAndSwapUint64((*uint64)(f), oldBits, newBits) {
			return
		}
	}
}

// Set sets the value.
func (f *AtomicFloat64) Set(value float64) {
	atomic.StoreUint64((*uint64)(f), math.Float64bits(value))
}

// Get returns current value.
func (f *AtomicFloat64) Get() float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(f)))
}

// Swap updates the value and returns the old one.
func (f *AtomicFloat64) Swap(value float64) float64 {
	return math.Float64frombits(atomic.SwapUint64((*uint64)(f), math.Float64bits(value)))
}
