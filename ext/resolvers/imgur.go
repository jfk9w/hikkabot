package resolvers

import (
	"bufio"
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/jfk9w-go/flu"

	httpf "github.com/jfk9w-go/flu/httpf"
	"github.com/pkg/errors"
)

var imgurRegexp = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*?)".*`)

type Imgur string

func (r *Imgur) GetClient(defaultClient httpf.Client) httpf.Client {
	return defaultClient
}

func (r *Imgur) Resolve(ctx context.Context, client httpf.Client, url string, _ int64) (string, error) {
	if *r != "" {
		return string(*r), nil
	}

	if err := httpf.GET(url).
		Exchange(ctx, client).
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
					*r = Imgur(groups[2])
					return nil
				}
			}

			return errors.New("unable to find URL")
		}).
		Error(); err != nil {
		return url, nil
	}

	return (string)(*r), nil
}
