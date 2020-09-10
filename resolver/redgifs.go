package resolver

import (
	"context"
	"net/http"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type RedGIFs struct {
	URL string
}

func (r *RedGIFs) GetClient() *fluhttp.Client {
	return nil
}

func (r *RedGIFs) ResolveURL(ctx context.Context, client *fluhttp.Client, url string, _ int64) (string, error) {
	if err := client.GET(url).
		Context(ctx).
		Execute().
		CheckStatus(http.StatusOK).
		HandleResponse(r).
		Error; err != nil {
		return "", errors.Wrap(err, "get html")
	}

	return r.URL, nil
}

func (r *RedGIFs) Request(request *fluhttp.Request) *fluhttp.Request {
	return request
}

func (r *RedGIFs) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)
	for tokenizer.Next() != html.ErrorToken {
		token := tokenizer.Token()
		if token.Type == html.StartTagToken &&
			token.Data == "script" &&
			format.HTMLAttributes(token.Attr).Get("type") == "application/ld+json" {
			if tokenizer.Next() != html.ErrorToken {
				break
			}

			token := tokenizer.Token()
			if token.Type == html.TextToken {
				var data struct {
					Video struct {
						ContentURL string `json:"contentUrl"`
					} `json:"video"`
				}

				if err := flu.DecodeFrom(flu.Bytes(token.Data), flu.JSON{&data}); err != nil {
					continue
				}

				if data.Video.ContentURL != "" {
					r.URL = data.Video.ContentURL
					return nil
				}
			}
		}
	}

	return errors.New("unable to find URL")
}
