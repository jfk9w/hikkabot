package reddit

import (
	"regexp"

	"github.com/jfk9w-go/flu"
)

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
	return nil, ErrNoCanonicalURL
}
