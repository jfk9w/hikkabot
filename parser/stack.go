package parser

type stack struct {
	tags    []tag
	size    int
	onWrite func(part string)
}

func newStack(onWrite func(part string)) *stack {
	return &stack{
		onWrite: onWrite,
		tags:    make([]tag, 20),
		size:    0,
	}
}

func (s *stack) isEmpty() bool {
	return s.size == 0
}

func (s *stack) push(t tag) {
	if s.size == 20 {
		panic("push() on full stack")
	}

	for _, ee := range s.tags {
		if ee.token == t.token {
			t.token = ""
			break
		}
	}

	if s.contents() {
		s.onWrite(t.token)
	}

	t.contents = t.contents && s.contents()
	s.tags[s.size] = t
	s.size++
}

func (s *stack) peek() tag {
	if s.isEmpty() {
		panic("peek() on empty stack")
	}

	return s.tags[s.size-1]
}

func (s *stack) pop() tag {
	if s.isEmpty() {
		panic("pop() on empty stack")
	}

	s.size--
	t := s.tags[s.size]
	if t.close {
		s.onWrite(t.token)
	}

	return t
}

func (s *stack) contents() bool {
	if s.isEmpty() {
		return true
	} else {
		return s.peek().contents
	}
}
