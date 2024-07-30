package redditsave

import (
	"errors"
	"net/http"

	"github.com/jfk9w-go/flu"
	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"golang.org/x/net/html"
)

type resolveResponse struct {
	url string
}

func (h *resolveResponse) Handle(resp *http.Response) error {
	defer flu.CloseQuietly(resp.Body)
	tokenizer := html.NewTokenizer(resp.Body)
	for tokenizer.Next() != html.ErrorToken {
		token := tokenizer.Token()
		if token.Type == html.StartTagToken && token.Data == "a" {
			if tghtml.Get(token.Attr, "class") == "downloadbutton" {
				h.url = tghtml.Get(token.Attr, "href")
				return nil
			}
		}
	}

	return errors.New("unable to find url")
}
