package mediator

import (
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

var (
	maxPhotoSize = [2]int64{5 << 20, 10 << 20}
	maxMediaSize = [2]int64{20 << 20, 50 << 20}
)

func MaxSize(typ telegram.MediaType) [2]int64 {
	if typ == telegram.Photo {
		return maxPhotoSize
	} else {
		return maxMediaSize
	}
}

type OCR struct {
	Filtered  bool
	Languages []string
	Regexp    *regexp.Regexp
}

type Metadata struct {
	URL       string
	Size      int64
	Format    string
	ForceLoad bool
	OCR       OCR
}

func (m *Metadata) Handle(resp *http.Response) error {
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse Content-Length header")
	}
	m.Size = size
	return nil
}

type Request interface {
	Metadata() (*Metadata, error)
	flu.Readable
}

type HTTPRequest struct {
	URL    string
	Format string
	OCR    OCR
	body   io.Reader
}

func (r *HTTPRequest) Handle(resp *http.Response) error {
	r.body = resp.Body
	return nil
}

func (r *HTTPRequest) Metadata() (*Metadata, error) {
	metadata := &Metadata{
		URL:    r.URL,
		Format: r.Format,
	}
	return metadata, CommonClient.HEAD(r.URL).
		Execute().
		HandleResponse(metadata).
		Error
}

func (r *HTTPRequest) Reader() (io.Reader, error) {
	return CommonClient.GET(r.URL).Execute().Reader()
}

type DoneRequest struct {
	flu.Readable
	Metadata_ *Metadata
}

func (r *DoneRequest) Metadata() (*Metadata, error) {
	return r.Metadata_, nil
}

type FailedRequest struct {
	Error error
}

func (r *FailedRequest) Metadata() (*Metadata, error) {
	return nil, r.Error
}

func (r *FailedRequest) Reader() (io.Reader, error) {
	return nil, r.Error
}

type Future struct {
	URL  string
	req  Request
	res  *telegram.Media
	err  error
	work sync.WaitGroup
}

func (f *Future) Result() (*telegram.Media, error) {
	f.work.Wait()
	return f.res, f.err
}

var CommonClient = flu.NewClient(nil).
	AcceptResponseCodes(http.StatusOK).
	Timeout(5 * time.Minute)
