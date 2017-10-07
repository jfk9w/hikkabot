package parser

import (
	"github.com/jfk9w/tele2ch/dvach"
	"bytes"
	"strings"
	"golang.org/x/net/html"
	"errors"
)

const maxMessageLength = 4000

func Parse(post dvach.Post) []string {
	reader := strings.NewReader(post.Comment)
	tokenizer := html.NewTokenizer(reader)

	buf := new(bytes.Buffer)
	stack := newStack()
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.Data == "br" {
				var msg string
				msg, stack = stack.drain()
				buf.WriteString(msg)
				buf.WriteRune('\n')
				continue
			}

			t := start(token)
			stack.push(t)

		case html.TextToken:
			if stack.contents() {
				lines := strings.Split(token.Data, `\n`)
				if len(lines) == 1 {
					stack.write(escape(lines[0]))
				} else {
					first := true
					for _, line := range lines {
						if first {
							first = false
						} else {
							buf.WriteRune('\n')
						}

						stack.write(escape(line))
						var msg string
						msg, stack = stack.drain()
						buf.WriteString(msg)
					}
				}
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

var escaper = strings.NewReplacer(`\r`, ``, `\`, `\\`, `[`, `\[`, `]`, `\]`, `_`, `\_`, `*`, `\*`)
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
