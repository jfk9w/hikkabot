package request

import (
	"io"
	"net/http"
	"regexp"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w/hikkabot/mediator"
)

var gfycatre = regexp.MustCompile(`(?i)https://[a-z]+.gfycat.com/[a-z0-9]+?\.mp4`)

type Gfycat struct {
	URL     string
	realURL string
}

func (r *Gfycat) Metadata() (*mediator.Metadata, error) {
	response := flu.NewBuffer()
	err := mediator.CommonClient.
		GET(r.URL).
		Execute().
		CheckStatusCode(http.StatusOK).
		ReadBodyTo(response).
		Error
	if err != nil {
		return nil, errors.New("get")
	}
	r.realURL = string(gfycatre.Find(response.Bytes()))
	if r.realURL == "" {
		return nil, errors.New("unable to find URL")
	}
	metadata := &mediator.Metadata{
		URL:    r.realURL,
		Format: "mp4",
	}
	return metadata, mediator.CommonClient.
		HEAD(r.realURL).
		Execute().
		CheckStatusCode(http.StatusOK).
		HandleResponse(metadata).
		Error
}

func (r *Gfycat) Reader() (io.Reader, error) {
	return mediator.CommonClient.
		GET(r.realURL).
		Execute().
		Reader()
}
