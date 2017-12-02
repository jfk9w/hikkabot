package html2md

import (
	"strings"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/util"
	"golang.org/x/net/html"
)

func Parse(post dvach.Post) ([]string, error) {
	var (
		tokenizer = html.NewTokenizer(strings.NewReader(post.Comment))
		ctx       = newContext()
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
	attach := parseAttachments(post)

	l := util.MinInt(len(messages), len(attach))
	for i := 0; i < l; i++ {
		messages[i] = attach[i] + "\n" + messages[i]
	}

	if messages != nil {
		id := "#P" + post.Num + " /"
		if attach != nil {
			id += " "
		} else {
			id += "\n"
		}

		messages[0] = id + messages[0]
	}

	return messages, nil
}

func parseAttachments(post dvach.Post) []string {
	attach := make([]string, len(post.Files))
	for i, file := range post.Files {
		attach[i] = "[(A)](" + file.URL() + ")"
	}

	return attach
}
