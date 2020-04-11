package media

import (
	"io"
	"net/http"
	"strconv"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

type Descriptor interface {
	flu.Input
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
	Client fluhttp.Client
	URL    string
}

func (d URLDescriptor) Metadata(_ int64) (*Metadata, error) {
	h := new(metadataHEADResponseHandler)
	h.URL = d.URL
	return &h.Metadata, d.Client.HEAD(d.URL).
		Execute().
		CheckStatus(http.StatusOK).
		HandleResponse(h).
		Error
}

func (d URLDescriptor) Reader() (io.Reader, error) {
	return d.Client.GET(d.URL).Execute().
		CheckStatus(http.StatusOK).
		Reader()
}

type LocalDescriptor struct {
	Metadata_ *Metadata
	Resource
}

func (d LocalDescriptor) Metadata(_ int64) (*Metadata, error) {
	return d.Metadata_, nil
}
