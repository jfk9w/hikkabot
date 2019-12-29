package media

import (
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

const MinMediaSize int64 = 10 << 10

var MaxMediaSize = map[telegram.MediaType]int64{
	telegram.Photo: 10 * (2 << 20),
	telegram.Video: 50 * (2 << 20),
}

type SizeAwareReadable interface {
	flu.Readable
	Size() (int64, error)
}

type HTTPRequestReadable struct {
	Request *flu.Request
	size    int64
	body    io.Reader
	done    bool
}

func (r *HTTPRequestReadable) ensure() (err error) {
	if !r.done {
		err = r.Request.Send().HandleResponse(r).Error
	}
	return
}

func (r *HTTPRequestReadable) Handle(resp *http.Response) error {
	contentLength := resp.Header.Get("Content-Length")
	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse Content-Length header")
	}
	r.size = size
	r.body = resp.Body
	return nil
}

func (r *HTTPRequestReadable) Reader() (reader io.Reader, err error) {
	err = r.ensure()
	if err != nil {
		return
	}
	reader = r.body
	return
}

func (r *HTTPRequestReadable) Size() (size int64, err error) {
	err = r.ensure()
	if err != nil {
		return
	}
	size = r.size
	return
}

type Media struct {
	URL    string
	format string
	in     SizeAwareReadable
	out    *TypeAwareReadable
	err    error
	work   sync.WaitGroup
}

type TypeAwareReadable struct {
	flu.Readable
	Type telegram.MediaType
}

func (m *Media) Ready() (*TypeAwareReadable, error) {
	m.work.Wait()
	return m.out, m.err
}
