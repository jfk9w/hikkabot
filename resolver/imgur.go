package resolver

import (
	"bufio"
	"context"
	"net/http"
	"regexp"
	"strings"

	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/pkg/errors"
)

var ImgurRegexp = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*?)".*`)

type Imgur struct {
	URL string
}

func (r *Imgur) GetClient() *fluhttp.Client {
	return nil
}

func (r *Imgur) ResolveURL(ctx context.Context, client *fluhttp.Client, url string, maxSize int64) (string, error) {
	if err := client.GET(url).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		HandleResponse(r).
		Error; err != nil {
		return url, nil
	} else {
		return r.URL, nil
	}
}

func (r *Imgur) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return errors.New("not an html")
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		groups := ImgurRegexp.FindStringSubmatch(line)
		if len(groups) == 3 {
			r.URL = groups[2]
			return nil
		}
	}

	return errors.New("unable to find URL")
}
