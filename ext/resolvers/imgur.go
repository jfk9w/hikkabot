package resolvers

import (
	"bufio"
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"hikkabot/feed/media"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

var imgurRegexp = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*?)".*`)

type Imgur[C any] struct{}

func (r Imgur[C]) String() string {
	return "media-resolver.imgur"
}

func (r *Imgur[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	return nil
}

func (r *Imgur[C]) Resolve(ctx context.Context, source *url.URL) (media.MetaRef, error) {
	switch source.Host {
	case "imgur.com", "www.imgur.com", "i.imgur.com", "m.imgur.com":
	default:
		return nil, nil
	}

	url := source.String()
	if strings.HasSuffix(url, ".jpg") {
		return &media.HTTPRef{URL: url}, nil
	}

	if strings.Contains(url, ".gifv") {
		url := strings.Replace(url, ".gifv", ".mp4", 1)
		return &media.HTTPRef{URL: url}, nil
	}

	var ref media.HTTPRef
	return &ref, httpf.GET(url).
		Exchange(ctx, nil).
		CheckStatus(http.StatusOK).
		HandleFunc(func(resp *http.Response) error {
			defer flu.CloseQuietly(resp.Body)
			contentType := resp.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "text/html") {
				return errors.New("not an html")
			}

			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				groups := imgurRegexp.FindStringSubmatch(line)
				if len(groups) == 3 {
					ref.URL = groups[2]
					return nil
				}
			}

			return errors.New("unable to find URL")
		}).Error()
}
