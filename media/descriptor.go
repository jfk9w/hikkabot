package media

import (
	"io"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type Descriptor interface {
	flu.Readable
	Metadata(maxSize int64) (*Metadata, error)
}

type metadataHEADResponseHandler struct {
	Metadata
}

func (m *metadataHEADResponseHandler) Handle(resp *http.Response) error {
	var err error
	m.Size, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return errors.Wrap(err, "parse Content-Length header")
	}

	m.MIMEType = resp.Header.Get("Content-Type")
	return err
}

type URLDescriptor struct {
	Client *flu.Client
	URL    string
}

func (d URLDescriptor) Metadata(_ int64) (*Metadata, error) {
	h := new(metadataHEADResponseHandler)
	h.URL = d.URL
	return &h.Metadata, d.Client.HEAD(d.URL).
		Execute().
		CheckStatusCode(http.StatusOK).
		HandleResponse(h).
		Error
}

func (d URLDescriptor) Reader() (io.Reader, error) {
	return d.Client.GET(d.URL).Execute().
		CheckStatusCode(http.StatusOK).
		Reader()
}
