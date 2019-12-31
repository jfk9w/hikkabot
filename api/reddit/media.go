package reddit

import (
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type ResolvedMedia struct {
	URL       string
	Container string
}

type MediaResolver interface {
	Resolve(*flu.Client, *Thing) (*ResolvedMedia, error)
}

var DefaultMediaResolver MediaResolver = plainMediaResolver{}

type plainMediaResolver struct{}

func (plainMediaResolver) Resolve(http *flu.Client, thing *Thing) (*ResolvedMedia, error) {
	url := thing.Data.URL
	idx := strings.LastIndex(url, ".")
	media := &ResolvedMedia{URL: url}
	if idx > 0 {
		media.Container = url[idx+1:]
	}
	return media, nil
}

func AddMediaScanner(domain string, scanner MediaResolver) {
	if _, ok := mediaResolvers[domain]; ok {
		panic(errors.Errorf("media scanner for %s already exists", domain, scanner))
	}
	mediaResolvers[domain] = scanner
}

var mediaResolvers = map[string]MediaResolver{
	"i.imgur.com": DefaultMediaResolver,
	"vidble.com":  DefaultMediaResolver,
	"i.redd.it":   ireddit(`^.*\.(.*)$`),
	"v.redd.it":   vRedditMediaResolver{},
	"imgur.com":   imgur(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*\.(.*?))".*`),
	"gfycat.com":  gfycat(`(?i)https://[a-z]+.gfycat.com/[a-z0-9]+?\.mp4`),
	"youtube.com": YoutubeMediaResolver{},
	"youtu.be":    YoutubeMediaResolver{},
}

var ErrNoCanonicalURL = errors.New("unable to find canonical URL")
