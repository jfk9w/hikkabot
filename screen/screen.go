package screen

import (
	"strings"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/util"
	"github.com/jfk9w/hikkabot/webm"
	"golang.org/x/net/html"
)

func Parse(board string, post dvach.Post, webms map[string]chan string) ([]string, error) {
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
			ctx.text(board, token)
			break

		case html.EndTagToken:
			ctx.end(token)
		}
	}

	ctx.dump()
	messages := ctx.messages
	attach := parseAttachments(post, webms)

	messagesLength := len(messages)
	attachLength := len(attach)
	l := util.MinInt(messagesLength, attachLength)
	for i := 0; i < l; i++ {
		messages[i] = attach[i] + "\n" + messages[i]
	}

	for i := messagesLength; i < attachLength; i++ {
		messages = append(messages, attach[i])
	}

	if len(messages) > 0 {
		id := "#" + strings.ToUpper(board) + post.Num + " /"
		if len(attach) > 0 {
			id += " "
		} else {
			id += "\n"
		}

		messages[0] = id + messages[0]
	}

	return messages, nil
}

func parseAttachments(post dvach.Post, webms map[string]chan string) []string {
	if len(post.Files) == 0 {
		return nil
	}

	attach := make([]string, len(post.Files))
	for i, file := range post.Files {
		url := file.URL()
		var v string
		if w, ok := webms[url]; ok {
			v = <-w
			if v == webm.Marked {
				v = url
			}
		} else {
			v = url
		}

		attach[i] = `<a href="` + escape(v) + `">[A]</a>`
	}

	return attach
}
