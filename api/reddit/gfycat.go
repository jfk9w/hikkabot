package reddit

import (
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type gfycatMediaScanner regexp.Regexp

func gfycat(re string) *gfycatMediaScanner {
	return (*gfycatMediaScanner)(regexp.MustCompile(re))
}

func (re *gfycatMediaScanner) Get(http *flu.Client, url string) (*Media, error) {
	media := new(Media)
	return media, http.NewRequest().
		GET().
		Resource(url).
		Send().
		HandleResponse(gfycatResponseHandler{media, (*regexp.Regexp)(re)}).
		Error
}

type gfycatResponseHandler struct {
	media *Media
	re    *regexp.Regexp
}

func (h gfycatResponseHandler) Handle(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return flu.StatusCodeError{resp.StatusCode, resp.Status}
	}
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return errors.Wrap(err, "on body read")
	}
	match := string(h.re.Find(data))
	if match != "" {
		h.media.URL = match
		h.media.Container = "mp4"
		return nil
	}
	return ErrNoCanonicalURL
}
