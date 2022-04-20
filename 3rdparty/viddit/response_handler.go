package viddit

import (
	"errors"
	"net/http"

	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"golang.org/x/net/html"
)

type resolveResponse struct {
	url string
}

func (h *resolveResponse) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)
	for tokenizer.Next() != html.ErrorToken {
		token := tokenizer.Token()
		if token.Type == html.StartTagToken && token.Data == "a" {
			if tghtml.Get(token.Attr, "id") == "dlbutton" {
				h.url = tghtml.Get(token.Attr, "href")
				return nil
			}
		}
	}

	return errors.New("unable to find url")
}
