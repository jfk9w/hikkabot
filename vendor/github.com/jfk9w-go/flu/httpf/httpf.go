// Package httpf provides HTTP request utilities similar to github.com/carlmjohnson/requests
// but with support of github.com/carlmjohnson/flu IO capabilities.
package httpf

import (
	"net/http"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// Authorization is used to set Authorization headers in requests.
type Authorization interface {
	SetAuth(req *http.Request)
}

// ContentType is an interface which may be optionally implemented by
// flu.EncoderTo implementations to automatically set Content-Type header
// in requests.
type ContentType interface {
	ContentType() string
}

// Client is a basic HTTP request executor.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// ResponseHandler handles http.Response.
type ResponseHandler interface {
	Handle(resp *http.Response) error
}

// ResponseHandlerFunc is ResponseHandler functional adapter.
type ResponseHandlerFunc func(resp *http.Response) error

func (f ResponseHandlerFunc) Handle(resp *http.Response) error {
	return f(resp)
}

// Catch handles exchange error.
type Catch interface {
	Handle(resp *http.Response, err error) error
}

// CatchFunc is Catch functional adapter.
type CatchFunc func(resp *http.Response, err error) error

func (f CatchFunc) Handle(resp *http.Response, err error) error {
	return f(resp, err)
}

// NewDefaultTransport creates a new http.Transport based on http.DefaultTransport.
func NewDefaultTransport() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}

// RoundTripperFunc is http.RoundTripper functional adapter.
type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Request creates a new HTTP request.
func Request(resource string) *RequestBuilder {
	req, err := http.NewRequest("", resource, nil)
	if err == nil {
		req.Method = ""
	}

	return &RequestBuilder{Request: req, err: errors.Wrap(err, "create http request")}
}

// GET creates a GET HTTP request.
func GET(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodGet)
}

// HEAD creates a HEAD HTTP request.
func HEAD(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodHead)
}

// POST creates a POST HTTP request.
func POST(resource string, body flu.EncoderTo) *RequestBuilder {
	return Request(resource).Method(http.MethodPost).Body(body)
}

// PUT creates a PUT HTTP request.
func PUT(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodPut)
}

// PATCH creates a PATCH HTTP request.
func PATCH(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodPatch)
}

// DELETE creates a DELETE HTTP request.
func DELETE(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodDelete)
}

// CONNECT creates a CONNECT HTTP request.
func CONNECT(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodConnect)
}

// OPTIONS creates an OPTIONS HTTP request.
func OPTIONS(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodOptions)
}

// TRACE creates a TRACE HTTP request.
func TRACE(resource string) *RequestBuilder {
	return Request(resource).Method(http.MethodTrace)
}
