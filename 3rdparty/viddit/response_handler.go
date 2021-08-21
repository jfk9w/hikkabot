package viddit

import (
	"errors"
	"net/http"

	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"

	"golang.org/x/net/html"
)

type responseHandler struct {
	url string
}

func (h *responseHandler) Handle(resp *http.Response) error {
	defer resp.Body.Close()
	tokenizer := html.NewTokenizer(resp.Body)
	for tokenizer.Next() != html.ErrorToken {
		token := tokenizer.Token()
		if token.Type == html.StartTagToken && token.Data == "a" {
			attrs := richtext.HTMLAttributes(token.Attr)
			if attrs.Get("id") == "dlbutton" {
				h.url = attrs.Get("href")
				return nil
			}
		}
	}

	return errors.New("unable to find url")
}
