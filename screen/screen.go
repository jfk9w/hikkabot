package screen

import (
	"strings"

	"fmt"
	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/util"
	"github.com/jfk9w/hikkabot/webm"
	"golang.org/x/net/html"
)

func Thread(board string, post dvach.Post) (string, bool) {
	messages := parseComment(board, post)
	preview := fmt.Sprintf(
		"%s / %d\n%s",
		writeLink("[T]", dvach.FormatThreadURL(board, post.Num)), post.PostsCount,
		messages[0])

	hasAttachment := false
	for _, file := range post.Files {
		if file.Type != dvach.WebmType {
			preview = writeLink("[A]", post.Files[0].URL()) + " / " + preview
			hasAttachment = true
			break
		}
	}

	return preview, hasAttachment
}

func Post(board string, post dvach.Post, webms map[string]chan string) ([]string, error) {
	messages := parseComment(board, post)
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

func parseComment(board string, post dvach.Post) []string {
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
	return ctx.messages
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

		attach[i] = writeLink("[A]", v)
	}

	return attach
}

func writeLink(title string, url string) string {
	return fmt.Sprintf("<a href=\"%s\">%s</a>", escape(url), title)
}
