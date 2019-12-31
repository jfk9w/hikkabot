package reddit

import (
	"errors"
	"regexp"

	"github.com/jfk9w-go/flu"
)

type iRedditMediaResolver regexp.Regexp

func ireddit(re string) *iRedditMediaResolver {
	return (*iRedditMediaResolver)(regexp.MustCompile(re))
}

func (re *iRedditMediaResolver) Resolve(_ *flu.Client, thing *Thing) (*ResolvedMedia, error) {
	url := thing.Data.URL
	groups := (*regexp.Regexp)(re).FindStringSubmatch(url)
	media := &ResolvedMedia{URL: url}
	if len(groups) == 2 {
		media.Container = groups[1]
		return media, nil
	}
	return nil, ErrNoCanonicalURL
}

type vRedditMediaResolver struct{}

func (r vRedditMediaResolver) Resolve(_ *flu.Client, thing *Thing) (*ResolvedMedia, error) {
	url := r.getURL(thing.Data.MediaContainer)
	if url == "" {
		for _, mc := range thing.Data.CrosspostParentList {
			url = r.getURL(mc)
			if url != "" {
				break
			}
		}
	}
	if url == "" {
		return nil, errors.New("no fallback URL")
	} else {
		return &ResolvedMedia{url, "mp4"}, nil
	}
}

func (r vRedditMediaResolver) getURL(mc MediaContainer) string {
	url := mc.Media.RedditVideo.FallbackURL
	if url == "" {
		url = mc.SecureMedia.RedditVideo.FallbackURL
	}
	return url
}
