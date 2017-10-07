package parser

import (
	"errors"
	"fmt"
	"github.com/jfk9w/tele2ch/dvach"
	"golang.org/x/net/html"
	"strings"
)

const maxMessageLength = 3900

func Parse(post dvach.Post) []string {
	reader := strings.NewReader(post.Comment)
	tokenizer := html.NewTokenizer(reader)

	lines := make([]part, 0)
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
				var line part
				line, stack = stack.drain()
				lines = append(lines, line)
				continue
			}

			t := start(token)
			stack.push(t)

		case html.TextToken:
			if stack.contents() {
				ls := strings.Split(token.Data, `\n`)
				if len(ls) == 1 {
					stack.write(escape(ls[0]))
				} else {
					var line part
					for i, l := range ls {
						stack.write(escape(l))
						line, stack = stack.drain()

						if i != len(lines)-1 {
							lines = append(lines, line)
						}
					}
				}
			}

		case html.EndTagToken:
			for !stack.isEmpty() {
				t := stack.pop()
				if t.typ == token.Data {
					break
				}
			}
		}
	}

	line, _ := stack.drain()
	if len(line.text) > 0 {
		lines = append(lines, line)
	}

	reparted := repartition(lines)
	msgs := make([]string, len(reparted))
	fileCount := 0
	files := len(post.Files)
	for i, part := range reparted {
		if i == 0 {
			part.text = fmt.Sprintf("#P%s /\n%s", post.Num, part.text)
		}

		if fileCount < files {
			if !part.hasLink {
				f := post.Files[fileCount]
				msgs[i] = fmt.Sprintf("%s\n---\n%s", part.text, attach(f))
				fileCount++

				continue
			}
		}

		msgs[i] = part.text
	}

	for ; fileCount < files; fileCount++ {
		msgs = append(msgs, attach(post.Files[fileCount]))
	}

	return msgs
}

func attach(file dvach.File) string {
	return fmt.Sprintf("[%s](%s)", escapeAttach(file.Name), escapeAttach(file.URL()))
}

func escapeAttach(value string) string {
	return strings.Replace(escape(value), `]`, `\]`, -1)
}

func repartition(lines []part) []part {
	parts := make([]part, 0)
	curr := part{}
	for _, line := range lines {
		test := curr.text
		if len(test) > 0 {
			test += "\n"
		}

		test += line.text
		if len(test) > maxMessageLength {
			parts = append(parts, curr)
			curr = part{}
		} else {
			curr.text = test
			curr.hasLink = curr.hasLink || line.hasLink
		}
	}

	if len(curr.text) > 0 {
		parts = append(parts, curr)
	}

	if len(parts) == 0 {
		parts = append(parts, part{})
	}

	return parts
}

var escaper = strings.NewReplacer(
	`\r`, ``,
	`\`, `\\`,
	`[`, `\[`,
	`_`, `\_`,
	`*`, `\*`)

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
