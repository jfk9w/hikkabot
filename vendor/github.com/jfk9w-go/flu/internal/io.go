package internal

import "io"

// Input interface describes a resource which can be read
// (possibly more than once).
type Input interface {
	// Reader returns an instance of io.Reader.
	Reader() (io.Reader, error)
}

// Output interface describes a resource which can be written
// (possibly more than once).
type Output interface {
	// Writer returns an instance of io.Writer.
	Writer() (io.Writer, error)
}

// IO is a generic Input / Output.
// It is not mandatory to fill all struct fields.
type IO struct {
	// R is an io.Reader instance to be used for reading.
	R io.Reader
	// W is an io.Writer instance to be used for reading.
	W io.Writer
	// E is an error which will be returned when reading/writing.
	E error
}

func (io IO) Reader() (io.Reader, error) {
	if io.E != nil {
		return nil, io.E
	}

	return io.R, nil
}

func (io IO) Writer() (io.Writer, error) {
	if io.E != nil {
		return nil, io.E
	}

	return io.W, nil
}
