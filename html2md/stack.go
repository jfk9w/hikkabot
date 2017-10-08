package html2md

import "bytes"

type stack struct {
	tags    []tag
	size    int
	buf     *bytes.Buffer
}

func newStack() *stack {
	return &stack{
		tags: make([]tag, 0),
		size: 0,
		buf:  new(bytes.Buffer),
	}
}

func (s *stack) isEmpty() bool {
	return s.size == 0
}

func (s *stack) push(t tag) {
	for _, ee := range s.tags {
		if ee.token == t.token {
			t.token = ""
			break
		}
	}

	if s.contents() {
		s.write(t.token)
	}

	t.contents = t.contents && s.contents()
	s.tags = append(s.tags, t)
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
		s.write(t.token)
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

func (s *stack) drain() (string, *stack) {
	next := newStack()
	for ; !s.isEmpty(); {
		t := s.pop()
		next.push(t)
	}

	return s.buf.String(), next
}

func (s *stack) write(text string) {
	s.buf.WriteString(text)
}
