package html

import (
	"regexp"
	"strings"

	"github.com/jfk9w-go/dvach"
	"golang.org/x/net/html"
)

type Post struct {
	dvach.Post
	Board        string
	Hash         string
	PostsPerHour float64
}

var (
	spanr = regexp.MustCompile(`<span.*?>`)
	tagr  = strings.NewReplacer(
		"<br>", "\n",
		"<strong>", "<b>",
		"</strong>", "</b>",
		"<em>", "<i>",
		"</em>", "</i>",
		"</span>", "</i>",
	)
)

func Chunk(post Post, chunkSize int, isThread bool) []string {
	var (
		text      = string(spanr.ReplaceAll([]byte(tagr.Replace(post.Comment)), []byte("<i>")))
		reader    = strings.NewReader(text)
		tokenizer = html.NewTokenizer(reader)
		builder   = NewBuilder(chunkSize)
		skip      = false
	)

	num := Num(post.Board, post.NumString)
	if isThread {
		builder.WriteThreadHeader(num, post.Subject, post.PostsCount, post.PostsPerHour)
	} else {
		if post.ParentString == "0" {
			builder.WriteMark()
		}

		builder.WritePostHeader(num, post.Hash)
	}

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
					builder.WriteNum(Num(post.Board, datanum))
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

			skip = false
		}
	}

	return builder.Done()
}

func Escape(value string) string {
	return html.EscapeString(value)
}

func Num(board, num string) string {
	return "#" + strings.ToUpper(board) + num
}

func attr(token html.Token, key string) (string, bool) {
	for _, attr := range token.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}

	return "", false
}
