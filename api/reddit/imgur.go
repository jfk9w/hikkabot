package reddit

import (
	"bufio"
	"net/http"
	"regexp"

	"github.com/jfk9w-go/flu"
)

type imgurMediaScanner regexp.Regexp

func imgur(re string) *imgurMediaScanner {
	return (*imgurMediaScanner)(regexp.MustCompile(re))
}

func (re *imgurMediaScanner) Get(http *flu.Client, url string) (*Media, error) {
	media := new(Media)
	return media, http.NewRequest().
		GET().
		Resource(url).
		Send().
		HandleResponse(imgurResponseHandler{media, (*regexp.Regexp)(re)}).
		Error
}

type imgurResponseHandler struct {
	media *Media
	re    *regexp.Regexp
}

func (d imgurResponseHandler) Handle(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return flu.StatusCodeError{resp.StatusCode, resp.Status}
	}
	scanner := bufio.NewScanner(resp.Body)
	defer resp.Body.Close()
	for scanner.Scan() {
		line := scanner.Text()
		groups := d.re.FindStringSubmatch(line)
		if len(groups) == 4 {
			d.media.URL = groups[2]
			d.media.Container = groups[3]
			return nil
		}
	}
	return ErrNoCanonicalURL
}
