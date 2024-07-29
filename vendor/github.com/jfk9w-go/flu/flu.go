package flu

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/jfk9w-go/flu/internal"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
)

var (
	ID       = internal.ID
	Readable = internal.Readable
)

// EncoderTo interface describes a value which can be encoded.
type EncoderTo interface {
	// EncodeTo encodes the value to the given io.Writer.
	EncodeTo(io.Writer) error
}

// DecoderFrom interface describes a value which can be decoded.
type DecoderFrom interface {
	// DecodeFrom decodes the value from the given io.Reader.
	DecodeFrom(io.Reader) error
}

// ValueCodec is a value container with encoder and decoder.
type ValueCodec interface {
	EncoderTo
	DecoderFrom
}

// Codec creates a ValueCodec for a given value.
type Codec func(interface{}) ValueCodec

// EncodeTo encodes the provided EncoderTo to Output.
// It closes the io.Writer instance if necessary.
func EncodeTo(encoder EncoderTo, out Output) error {
	w, err := out.Writer()
	if err != nil {
		return err
	}
	if err := encoder.EncodeTo(w); err != nil {
		return err
	}
	return Close(w)
}

// DecodeFrom decodes the provided DecoderFrom from Input.
// It closes the io.Reader instance if necessary.
func DecodeFrom(in Input, decoder DecoderFrom) error {
	r, err := in.Reader()
	if err != nil {
		return err
	}
	if err := decoder.DecodeFrom(r); err != nil {
		return err
	}
	return Close(r)
}

// PipeInput pipes the encoded value from EncoderTo as Input
// in the background.
func PipeInput(encoder EncoderTo) Input {
	r, w := io.Pipe()
	_, _ = syncf.Go(context.Background(), func(ctx context.Context) {
		if err := w.CloseWithError(encoder.EncodeTo(w)); err != nil {
			logf.Warnf(ctx, "close pipe writer error: %v", err)
		}
	})

	return IO{R: r}
}

// PipeOutput provides an Output which feeds into DecoderFrom
// in the background.
func PipeOutput(decoder DecoderFrom) Output {
	r, w := io.Pipe()
	_, _ = syncf.Go(context.Background(), func(ctx context.Context) {
		if err := r.CloseWithError(decoder.DecodeFrom(r)); err != nil {
			logf.Warnf(ctx, "close pipe reader error: %v", err)
		}
	})

	return IO{W: w}
}

// Copy copies the Input to the Output.
func Copy(in Input, out Output) (written int64, err error) {
	r, err := in.Reader()
	if err != nil {
		return
	}

	defer CloseQuietly(r)

	w, err := out.Writer()
	if err != nil {
		return
	}

	defer CloseQuietly(w)

	return io.Copy(w, r)
}

// ToString reads an Input to a string.
func ToString(in Input) (string, error) {
	buf := new(ByteBuffer)
	_, err := Copy(in, buf)
	if err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}

// Sleep sleeps for the specified timeout interruptibly.
func Sleep(ctx context.Context, timeout time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		return nil
	}
}

// Counter is an int64 counter.
type Counter int64

// Value returns the current value of the *Counter.
func (c *Counter) Value() int64 {
	return *(*int64)(c)
}

// Add adds an int64 value to the counter.
func (c *Counter) Add(n int64) {
	*(*int64)(c) += n
}

// Close attempts to close the provided value
// using io.Closer interface.
func Close(value any) error {
	if closer, ok := value.(io.Closer); ok {
		switch closer {
		case os.Stdin, os.Stdout, os.Stderr:
			return nil
		default:
			return closer.Close()
		}
	} else {
		return nil
	}
}

// CloseQuietly attempts to close the provided value using
// io.Closer with logging on error.
func CloseQuietly(values ...any) {
	for _, value := range values {
		if err := Close(value); err != nil {
			logf.Get(Readable(value)).Warnf(nil, "close error: %v", err)
		}
	}
}
