package descriptor

import (
	"bufio"
	"io"
	"net/http"
	"regexp"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
	"github.com/pkg/errors"
)

var imgurre = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*?)".*`)

type Imgur struct {
	Client *flu.Client
	URL    string
	urld   media.URLDescriptor
}

func (d *Imgur) Metadata(maxSize int64) (*media.Metadata, error) {
	if d.urld.URL == "" {
		h := new(imgurHTMLHandler)
		if err := d.Client.GET(d.URL).Execute().
			CheckStatusCode(http.StatusOK).
			HandleResponse(h).
			Error; err != nil {
			return nil, errors.Wrap(err, "get html")
		}

		d.urld.URL = h.URL
		d.urld.Client = d.Client
	}

	return d.urld.Metadata(maxSize)
}

func (d *Imgur) Reader() (io.Reader, error) {
	return d.urld.Reader()
}

type imgurHTMLHandler struct {
	URL string
}

func (h *imgurHTMLHandler) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		groups := imgurre.FindStringSubmatch(line)
		if len(groups) == 3 {
			h.URL = groups[2]
			return nil
		}
	}

	return errors.New("unable to find URL")
}
