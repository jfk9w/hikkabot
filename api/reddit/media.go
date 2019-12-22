package reddit

import (
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

var ErrNoCanonicalURL = errors.New("unable to find canonical URL")
