package httpf

import (
	"io"
	"net/http"
	"strings"

	"github.com/jfk9w-go/flu"
)

// ExchangeResult is a fluent response wrapper.
type ExchangeResult struct {
	*http.Response
	err error
}

// ExchangeError returns an ExchangeResult stub with error set.
func ExchangeError(err error) *ExchangeResult {
	return &ExchangeResult{err: err}
}

// Handle executes a ResponseHandler if no previous handling errors occurred.
func (r *ExchangeResult) Handle(handler ResponseHandler) *ExchangeResult {
	if r.err != nil {
		return r
	}

	r.err = handler.Handle(r.Response)
	return r
}

// Catch processes and overwrites the error if it occurred.
func (r *ExchangeResult) Catch(catch Catch) *ExchangeResult {
	if r.err == nil {
		return r
	}

	r.err = catch.Handle(r.Response, r.err)
	return r
}

// HandleFunc is like Handle, but accepts functions.
func (r *ExchangeResult) HandleFunc(handler ResponseHandlerFunc) *ExchangeResult {
	return r.Handle(handler)
}

// CatchFunc is like Catch, but accepts functions.
func (r *ExchangeResult) CatchFunc(catch CatchFunc) *ExchangeResult {
	return r.Catch(catch)
}

// CheckStatus checks the response status code and sets the error to StatusCodeError if there is no match.
// It closes response body on failure.
func (r *ExchangeResult) CheckStatus(codes ...int) *ExchangeResult {
	return r.HandleFunc(func(resp *http.Response) error {
		for _, code := range codes {
			if resp.StatusCode == code {
				return nil
			}
		}

		return StatusCodeError{StatusCode: r.StatusCode, Status: r.Status}
	})
}

// CheckContentType checks the response Content-Type and returns error if it does not match the value.
// It closes response body on failure.
func (r *ExchangeResult) CheckContentType(value string) *ExchangeResult {
	return r.HandleFunc(func(resp *http.Response) error {
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, value) {
			return ContentTypeError(contentType)
		}

		return nil
	})
}

// DecodeBody decodes response body using the provided decoder.
func (r *ExchangeResult) DecodeBody(decoder flu.DecoderFrom) *ExchangeResult {
	if err := flu.DecodeFrom(r, decoder); err != nil && r.err == nil {
		r.err = err
	}

	return r
}

// Reader implements flu.Input interface.
func (r *ExchangeResult) Reader() (io.Reader, error) {
	var body io.Reader
	if resp := r.Response; resp != nil {
		body = resp.Body
	}

	return body, r.err
}

// CopyBody copies response body to flu.Output.
func (r *ExchangeResult) CopyBody(out flu.Output) *ExchangeResult {
	return r.HandleFunc(func(resp *http.Response) error {
		_, err := flu.Copy(flu.IO{R: resp.Body}, out)
		return err
	})
}

// Error terminates the exchange, closes response body if necessary, and returns an error if any.
func (r *ExchangeResult) Error() error {
	if resp := r.Response; resp != nil {
		flu.CloseQuietly(resp.Body)
	}

	return r.err
}
