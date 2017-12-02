package html2md

import (
	"bytes"
	"github.com/jfk9w/hikkabot/dvach"
	"golang.org/x/net/html"
	"strings"
)

func Parse(post dvach.Post) ([]string, error) {
	var (
		tokenizer = html.NewTokenizer(strings.NewReader(post.Comment))
		ctx = newContext()
	)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			ctx.start(token)
			break

		case html.TextToken:
			ctx.text(token)
			break

		case html.EndTagToken:
			ctx.end(token)
		}
	}

	ctx.dump()
	messages := ctx.messages

	messages[0] =
}
