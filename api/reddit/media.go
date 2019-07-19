package reddit

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type media struct {
	url string
	ext string
}

type mediaScanner func(*flu.Client, string) (*media, error)

var (
	redditCanonicalLinkRegexp = regexp.MustCompile(`^.*\.(.*)$`)
	imgurCanonicalLinkRegexp  = regexp.MustCompile(`.*?(<link rel="image_src"\s+href="|<meta property="og:video"\s+content=")(.*\.(.*?))".*`)
	gfycatCanonicalLinkRegexp = regexp.MustCompile(`(?i)https://[a-z]+.gfycat.com/[a-z0-9]+?\.mp4`)

	mediaScanners = map[string]mediaScanner{
		"i.redd.it": func(c *flu.Client, u string) (*media, error) {
			groups := redditCanonicalLinkRegexp.FindStringSubmatch(u)
			m := &media{url: u}
			if len(groups) == 2 {
				m.ext = groups[1]
				return m, nil
			}

			return nil, errors.New("unable to detect file format")
		},
		"i.imgur.com": func(c *flu.Client, u string) (*media, error) {
			idx := strings.LastIndex(u, ".")
			m := &media{url: u}
			if idx > 0 {
				m.ext = u[idx+1:]
			}

			return m, nil
		},
		"imgur.com": func(c *flu.Client, u string) (*media, error) {
			m := new(media)
			return m, c.NewRequest().
				GET().
				Resource(u).
				Send().
				ReadBodyFunc(func(body io.Reader) error {
					scanner := bufio.NewScanner(body)
					for scanner.Scan() {
						line := scanner.Text()
						groups := imgurCanonicalLinkRegexp.FindStringSubmatch(line)
						if len(groups) == 4 {
							m.url = groups[2]
							m.ext = groups[3]
							return nil
						}
					}

					return errors.New("unable to find canonical url")
				}).
				Error
		},
		"gfycat.com": func(c *flu.Client, u string) (*media, error) {
			m := new(media)
			return m, c.NewRequest().
				GET().
				Resource(u).
				Send().
				ReadBytesFunc(func(data []byte) error {
					match := string(gfycatCanonicalLinkRegexp.Find(data))
					if match != "" {
						m.url = match
						m.ext = "mp4"
						return nil
					}

					return errors.New("unable to find canonical url")
				}).
				Error
		},
	}
)
