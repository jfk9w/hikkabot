package flu

import (
	"io"
)

// ReaderCounter is a counting io.Reader.
// Useful for calculating the total size of read data.
type ReaderCounter struct {
	io.Reader
	*Counter
}

func (rc ReaderCounter) Read(data []byte) (n int, err error) {
	n = len(data)
	if rc.Reader != nil {
		n, err = rc.Reader.Read(data)
		if err != nil {
			return
		}
	}

	rc.Add(int64(n))
	return
}

func (rc ReaderCounter) Close() error {
	return Close(rc.Reader)
}

// WriterCounter is a counting io.Writer.
// Useful for calculating the total size of written data.
type WriterCounter struct {
	io.Writer
	*Counter
}

func (wc WriterCounter) Write(data []byte) (n int, err error) {
	n = len(data)
	if wc.Writer != nil {
		n, err = wc.Writer.Write(data)
		if err != nil {
			return 0, err
		}
	}

	wc.Add(int64(n))
	return n, nil
}

func (wc WriterCounter) Close() error {
	return Close(wc.Writer)
}

// IOCounter is a counting wrapper for Input and/or Output.
type IOCounter struct {
	Input
	Output
	Counter
}

func (c *IOCounter) Reader() (r io.Reader, err error) {
	if c.Input != nil {
		r, err = c.Input.Reader()
		if err != nil {
			return nil, err
		}
	}

	return ReaderCounter{Reader: r, Counter: &c.Counter}, nil
}

func (c *IOCounter) Writer() (w io.Writer, err error) {
	if c.Output != nil {
		w, err = c.Output.Writer()
		if err != nil {
			return
		}
	}

	return WriterCounter{Writer: w, Counter: &c.Counter}, nil
}
