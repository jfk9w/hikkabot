package request

import (
	"bufio"
	"io"
	"net/http"
	"regexp"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/pkg/errors"
)

var imgurre = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*\.(.*?))".*`)

type Imgur struct {
	URL     string
	OCR     mediator.OCR
	realURL string
	format  string
}

func (r *Imgur) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		groups := imgurre.FindStringSubmatch(line)
		if len(groups) == 4 {
			r.realURL = groups[2]
			r.format = groups[3]
			return nil
		}
	}
	return errors.New("unable to find URL")
}

func (r *Imgur) Metadata() (*mediator.Metadata, error) {
	err := flu.DefaultClient.
		GET(r.URL).
		Execute().
		CheckStatusCode(http.StatusOK).
		HandleResponse(r).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "get")
	}
	metadata := &mediator.Metadata{
		URL:    r.realURL,
		Format: r.format,
		OCR:    r.OCR,
	}
	return metadata, mediator.CommonClient.
		HEAD(r.realURL).
		Execute().
		CheckStatusCode(http.StatusOK).
		HandleResponse(metadata).
		Error
}

func (r *Imgur) Reader() (io.Reader, error) {
	return mediator.CommonClient.
		GET(r.realURL).
		Execute().
		CheckStatusCode(http.StatusOK).
		Reader()
}
