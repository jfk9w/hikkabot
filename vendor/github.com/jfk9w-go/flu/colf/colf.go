// Package colf contains some commonly used collections abstractions.
package colf

type (
	// ForEach is an iteration function.
	ForEach[E any] func(element E) bool

	// Iterable can be iterated with a ForEach function.
	Iterable[E any] interface{ ForEach(forEach ForEach[E]) }

	// Addable is a mutable collection which can be added elements to.
	Addable[E any] interface{ Add(element E) }

	// Sizeable is a collection with a defined size.
	Sizeable interface{ Size() int }
)

// Keys returns a Set containing map keys.
func Keys[K comparable, V any](values map[K]V) Slice[K] {
	keys := make(Slice[K], 0, len(values))
	for key := range values {
		keys.Add(key)
	}

	return keys
}

// AddAll adds all elements from Iterable to Addable.
func AddAll[E any](appendable Addable[E], iterable Iterable[E]) {
	iterable.ForEach(func(element E) bool {
		appendable.Add(element)
		return true
	})
}

// ToSlice collects all elements from Iterable into a Slice.
func ToSlice[E any](iterable Iterable[E]) Slice[E] {
	var size int
	if sizeable, ok := iterable.(Sizeable); ok {
		size = sizeable.Size()
	}

	slice := make(Slice[E], 0, size)
	AddAll[E](&slice, iterable)
	return slice
}

// Contains checks if an element is in an Iterable.
func Contains[E comparable](iterable Iterable[E], value E) (match bool) {
	iterable.ForEach(func(element E) bool {
		if element == value {
			match = true
			return false
		}

		return true
	})

	return
}
