package parser

import (
	"github.com/jfk9w/tele2ch/dvach"
	"bytes"
	"strings"
	"golang.org/x/net/html"
	"errors"
)

type context struct {
	buf   *bytes.Buffer
	parts []string
	size  int
}

func Parse(post dvach.Post) []string {
	reader := strings.NewReader(post.Comment)
	tokenizer := html.NewTokenizer(reader)

	buf := new(bytes.Buffer)
	stack := newStack(func(text string) {
		buf.WriteString(text)
	})

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.Data == "br" {
				buf.WriteString("\n")
				continue
			}

			t := start(token)
			stack.push(t)

		case html.TextToken:
			if stack.contents() {
				buf.WriteString(escape(token.Data))
			}

		case html.EndTagToken:
			for ; !stack.isEmpty(); {
				t := stack.pop()
				if t.typ == token.Data {
					break
				}
			}
		}
	}

	return []string{buf.String()}
}

var escaper = strings.NewReplacer(`\n`, "\n", `\r`, "",
	`\`, `\\`, `[`, `\[`, `]`, `\]`, `_`, `\_`, `*`, `\*`)

func escape(text string) string {
	return escaper.Replace(text)
}

func attr(attrs []html.Attribute, name string) (string, error) {
	for _, a := range attrs {
		if a.Key == name {
			return a.Val, nil
		}
	}

	return "", errors.New("no '" + name + "' attribute")
}
