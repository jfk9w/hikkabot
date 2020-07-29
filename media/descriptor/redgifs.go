package descriptor

import (
	"io"
	"net/http"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w/hikkabot/media"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type Redgifs struct {
	Client fluhttp.Client
	URL    string
	urld   media.URLDescriptor
}

func (d *Redgifs) Metadata(maxSize int64) (*media.Metadata, error) {
	if d.urld.URL == "" {
		h := new(redgifsHTMLHandler)
		if err := d.Client.GET(d.URL).Execute().
			CheckStatus(http.StatusOK).
			HandleResponse(h).
			Error; err != nil {
			return nil, errors.Wrap(err, "get html")
		}

		d.urld.URL = h.URL
		d.urld.Client = d.Client
	}

	return d.urld.Metadata(maxSize)
}

func (d *Redgifs) Reader() (io.Reader, error) {
	return d.urld.Reader()
}

type redgifsHTMLHandler struct {
	URL string
}

func (h *redgifsHTMLHandler) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if token.Type == html.StartTagToken &&
			token.Data == "script" &&
			attr(token.Attr, "type") == "application/ld+json" {
			tt := tokenizer.Next()
			if tt == html.ErrorToken {
				break
			}
			token := tokenizer.Token()
			if token.Type == html.TextToken {
				var data struct {
					Video struct {
						ContentURL string `json:"contentUrl"`
					} `json:"video"`
				}

				if err := flu.DecodeFrom(flu.Bytes(token.Data), flu.JSON{&data}); err != nil {
					continue
				}

				if data.Video.ContentURL != "" {
					h.URL = data.Video.ContentURL
					return nil
				}
			}
		}
	}

	return errors.New("unable to find URL")
}

func attr(attrs []html.Attribute, name string) string {
	for _, attr := range attrs {
		if attr.Key == name {
			return attr.Val
		}
	}

	return ""
}
