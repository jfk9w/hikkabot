package httpf

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

// RequestBuilder provides methods for building a http.Request.
type RequestBuilder struct {
	*http.Request
	query url.Values
	body  flu.Input
	err   error
}

func (r RequestBuilder) String() string {
	url := *r.Request.URL
	if r.query != nil {
		url.RawQuery = r.query.Encode()
	}

	return r.Request.Method + " " + url.String()
}

// Method sets HTTP method for this request.
func (r *RequestBuilder) Method(value string) *RequestBuilder {
	if r.err != nil {
		return r
	}
	r.Request.Method = value
	return r
}

// Header set HTTP header for this request.
func (r *RequestBuilder) Header(key, value string) *RequestBuilder {
	if r.err != nil {
		return r
	}
	r.Request.Header.Set(key, value)
	return r
}

func (r *RequestBuilder) Headers(kvPairs ...string) *RequestBuilder {
	if r.err != nil {
		return r
	}
	kvLength := keyValuePairsLength(kvPairs)
	for i := 0; i < kvLength; i += 2 {
		k, v := kvPairs[i], kvPairs[i+1]
		r.Header(k, v)
	}
	return r
}

func (r *RequestBuilder) Auth(auth Authorization) *RequestBuilder {
	if r.err != nil {
		return r
	}
	auth.SetAuth(r.Request)
	return r
}

// Query sets a query parameter.
func (r *RequestBuilder) Query(key, value string) *RequestBuilder {
	if r.err != nil {
		return r
	}
	if r.query == nil {
		r.query = r.URL.Query()
	}
	r.query.Set(key, value)
	return r
}

// QueryValues sets query parameters from url.Values.
func (r *RequestBuilder) QueryValues(values url.Values) *RequestBuilder {
	if r.err != nil {
		return r
	}
	for key, values := range values {
		for _, value := range values {
			r.Query(key, value)
		}
	}
	return r
}

// ContentType sets Content-Type header for this request.
// If not set, Content-Type header will be set automatically if body implements ContentType interface.
func (r *RequestBuilder) ContentType(contentType string) *RequestBuilder {
	if r.err != nil {
		return r
	}
	return r.Header("Content-Type", contentType)
}

// ContentLength sets Content-Length header for this request.
// If not set, Content-Length header will be set automatically for request body io.Readers:
//   bytes.Buffer (flu.ByteBuffer)
//   bytes.Reader (flu.Bytes)
//   strings.Reader
// Note that some servers may not accept unknown content length.
func (r *RequestBuilder) ContentLength(contentLength int64) *RequestBuilder {
	if r.err != nil {
		return r
	}
	r.Request.ContentLength = contentLength
	return r
}

// Body is used to specify the request body as flu.EncoderTo.
// Note that if passed encoder implements ContentType interface,
// it will be used to set Content-Type header automatically.
func (r *RequestBuilder) Body(encoder flu.EncoderTo) *RequestBuilder {
	if r.err != nil {
		return r
	}

	if encoder == nil {
		r.body = nil
		return r
	}

	r.body = encoderInput{EncoderTo: encoder}
	return r
}

// BodyInput is used to specify the request body directly from flu.Input.
func (r *RequestBuilder) BodyInput(input flu.Input) *RequestBuilder {
	if r.err != nil {
		return r
	}
	r.body = input
	return r
}

// Exchange executes the request and returns a response.
func (r *RequestBuilder) Exchange(ctx context.Context, client Client) *ExchangeResult {
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := exchange(client, r.Request.Clone(ctx), r.query, r.body)
	return &ExchangeResult{resp, err}
}

func exchange(client Client, req *http.Request, query url.Values, body flu.Input) (*http.Response, error) {
	if req.URL == nil {
		return nil, errors.New("empty request url")
	}

	if req.Method == "" {
		if body == nil {
			req.Method = http.MethodGet
		} else {
			req.Method = http.MethodPost
		}
	}

	if body != nil {
		reader, err := body.Reader()
		if err != nil {
			return nil, errors.Wrap(err, "get body reader")
		}

		if body, ok := reader.(io.ReadCloser); ok {
			req.Body = body
		} else {
			req.Body = ioutil.NopCloser(body)
		}

		if req.ContentLength <= 0 {
			switch r := reader.(type) {
			case *bytes.Buffer:
				req.ContentLength = int64(r.Len())
			case *bytes.Reader:
				req.ContentLength = int64(r.Len())
			case *strings.Reader:
				req.ContentLength = int64(r.Len())
			}
		}

		if req.Header.Get("Content-Type") == "" {
			if ext, ok := body.(ContentType); ok && ext.ContentType() != "" {
				req.Header.Set("Content-Type", ext.ContentType())
			}
		}
	}

	if query != nil {
		req.URL.RawQuery = query.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type encoderInput struct {
	flu.EncoderTo
}

func (i encoderInput) Reader() (io.Reader, error) {
	return flu.PipeInput(i.EncoderTo).Reader()
}

func (i encoderInput) ContentType() string {
	if ct, ok := i.EncoderTo.(ContentType); ok {
		return ct.ContentType()
	}

	return ""
}

func keyValuePairsLength(kvPairs []string) int {
	length := len(kvPairs)
	if length%2 > 0 {
		panic(VarargsLengthError(length))
	}
	return length
}
