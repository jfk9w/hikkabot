package reddit

import (
	"bufio"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type Media struct {
	URL       string
	Container string
}

type MediaScanner interface {
	Get(*flu.Client, string) (*Media, error)
}

var DefaultMediaScanner MediaScanner = plainMediaScanner{}

type plainMediaScanner struct{}

func (plainMediaScanner) Get(http *flu.Client, url string) (*Media, error) {
	idx := strings.LastIndex(url, ".")
	media := &Media{URL: url}
	if idx > 0 {
		media.Container = url[idx+1:]
	}

	return media, nil
}

type redditMediaScanner regexp.Regexp

func reddit(re string) *redditMediaScanner {
	return (*redditMediaScanner)(regexp.MustCompile(re))
}

func (re *redditMediaScanner) Get(_ *flu.Client, url string) (*Media, error) {
	groups := (*regexp.Regexp)(re).FindStringSubmatch(url)
	media := &Media{URL: url}
	if len(groups) == 2 {
		media.Container = groups[1]
		return media, nil
	}

	return nil, errors.New("unable to detect file format")
}

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

func (d imgurResponseHandler) Handle(http *http.Response) error {
	if http.StatusCode != 200 {
		return flu.StatusCodeError{http.StatusCode, http.Status}
	}
	scanner := bufio.NewScanner(http.Body)
	defer http.Body.Close()
	for scanner.Scan() {
		line := scanner.Text()
		groups := d.re.FindStringSubmatch(line)
		if len(groups) == 4 {
			d.media.URL = groups[2]
			d.media.Container = groups[3]
			return nil
		}
	}
	return errors.New("unable to find canonical URL")
}

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

func (h gfycatResponseHandler) Handle(http *http.Response) error {
	if http.StatusCode != 200 {
		return flu.StatusCodeError{http.StatusCode, http.Status}
	}
	data, err := ioutil.ReadAll(http.Body)
	http.Body.Close()
	if err != nil {
		return errors.Wrap(err, "on body read")
	}
	match := string(h.re.Find(data))
	if match != "" {
		h.media.URL = match
		h.media.Container = "mp4"
		return nil
	}
	return errors.New("unable to find canonical URL")
}

func AddMediaScanner(domain string, scanner MediaScanner) {
	if _, ok := mediaScanners[domain]; ok {
		panic(errors.Errorf("media scanner for %s already exists", domain, scanner))
	}
	mediaScanners[domain] = scanner
}

var mediaScanners = map[string]MediaScanner{
	"i.imgur.com": DefaultMediaScanner,
	"vidble.com":  DefaultMediaScanner,
	"i.redd.it":   reddit(`^.*\.(.*)$`),
	"imgur.com":   imgur(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*\.(.*?))".*`),
	"gfycat.com":  gfycat(`(?i)https://[a-z]+.gfycat.com/[a-z0-9]+?\.mp4`),
}
