package descriptor

import (
	"io"
	"net/http"
	"regexp"

	fluhttp "github.com/jfk9w-go/flu/http"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w/hikkabot/media"
	"github.com/pkg/errors"
)

var gfycatre = regexp.MustCompile(`(?i)https://[a-z]+.gfycat.com/[a-z0-9]+?\.mp4`)

type Gfycat struct {
	Client fluhttp.Client
	URL    string
	urld   media.URLDescriptor
}

func (d *Gfycat) Metadata(maxSize int64) (*media.Metadata, error) {
	if d.urld.URL == "" {
		html := flu.NewBuffer()
		if err := d.Client.GET(d.URL).Execute().
			CheckStatus(http.StatusOK).
			DecodeBodyTo(html).
			Error; err != nil {
			return nil, errors.New("get html")
		}

		url := string(gfycatre.Find(html.Bytes()))
		if url == "" {
			return nil, errors.New("unable to find URL")
		}

		d.urld.URL = url
		d.urld.Client = d.Client
	}

	return d.urld.Metadata(maxSize)
}

func (d *Gfycat) Reader() (io.Reader, error) {
	return d.urld.Reader()
}
