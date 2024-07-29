package colf

// Slice is a wrapper for slices.
type Slice[E any] []E

func (s Slice[E]) ForEach(forEach ForEach[E]) {
	for _, element := range s {
		if !forEach(element) {
			break
		}
	}
}

func (s *Slice[E]) Add(element E) {
	if *s == nil {
		*s = make(Slice[E], 0)
	}

	*s = append(*s, element)
}

func (s Slice[E]) Size() int {
	return len(s)
}
