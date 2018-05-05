package html

import (
	"regexp"
	"strings"

	"github.com/jfk9w-go/dvach"
	"golang.org/x/net/html"
)

var (
	spanr = regexp.MustCompile(`<span.*>`)
	tagr  = strings.NewReplacer(
		"<br>", "\n",
		"<strong>", "<b>",
		"</strong>", "</b>",
		"<em>", "<i>",
		"</em>", "</i>",
		"</span>", "<i>",
	)
)

func Chunk(post dvach.Post, chunkSize int) []string {
	var (
		text      = string(spanr.ReplaceAll([]byte(tagr.Replace(post.Comment)), []byte("")))
		reader    = strings.NewReader(text)
		tokenizer = html.NewTokenizer(reader)
		builder   = NewBuilder(chunkSize)
		skip      = false
	)

	builder.WriteHashtag(num(post.Board, post.Num))
	builder.WriteText(" /\n")

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		data := token.Data
		switch tokenType {
		case html.StartTagToken:
			switch data {
			case "br":
				builder.WriteNewLine()

			case "a":
				if datanum, ok := attr(token, "data-num"); ok {
					builder.WriteHashtag(num(post.Board, datanum))
					skip = true
					continue
				}

				if link, ok := attr(token, "href"); ok {
					builder.WriteLink(link)
					skip = true
					continue
				}

			default:
				builder.WriteStartTag(token.String())
			}

		case html.TextToken:
			if !skip {
				builder.WriteText(data)
			}

		case html.EndTagToken:
			if !skip {
				builder.WriteEndTag()
			}
		}
	}

	return builder.Done()
}

func num(board, num string) string {
	return strings.ToUpper(board) + num
}

func attr(token html.Token, key string) (string, bool) {
	for _, attr := range token.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}

	return "", false
}
