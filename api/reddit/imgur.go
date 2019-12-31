package reddit

import (
	"bufio"
	"net/http"
	"regexp"

	"github.com/jfk9w-go/flu"
)

type imgurMediaResolver regexp.Regexp

func imgur(re string) *imgurMediaResolver {
	return (*imgurMediaResolver)(regexp.MustCompile(re))
}

func (re *imgurMediaResolver) Resolve(http *flu.Client, thing *Thing) (*ResolvedMedia, error) {
	media := new(ResolvedMedia)
	return media, http.NewRequest().
		GET().
		Resource(thing.Data.URL).
		Send().
		HandleResponse(imgurResponseHandler{media, (*regexp.Regexp)(re)}).
		Error
}

type imgurResponseHandler struct {
	media *ResolvedMedia
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
