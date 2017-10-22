package html2md

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jfk9w/hikkabot/dvach"
	"golang.org/x/net/html"
)

const maxMessageLength = 3900

func Parse(post dvach.Post) []string {
	reader := strings.NewReader(post.Comment)
	tokenizer := html.NewTokenizer(reader)

	lines := make([]string, 0)
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
				var line string
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
					var line string
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
	if len(line) > 0 {
		lines = append(lines, line)
	}

	reparted := repartition(lines)
	msgs := make([]string, len(reparted))
	fileCount := 0
	files := len(post.Files)
	for i, msg := range reparted {
		prefix := ""
		if i == 0 {
			prefix = fmt.Sprintf("#P%s /", post.Num)
		}

		if fileCount < files {
			f := post.Files[fileCount]
			if len(prefix) > 0 {
				prefix += " "
			}

			prefix += attach(f)
			if len(msg) > 0 {
				prefix += "\n"
			}

			fileCount++
		} else if i == 0 {
			prefix += "\n"
		}

		msgs[i] = prefix + msg
	}

	for ; fileCount < files; fileCount++ {
		msgs = append(msgs, attach(post.Files[fileCount]))
	}

	return msgs
}

func attach(file dvach.File) string {
	return fmt.Sprintf(`[(A)](%s)`, escapeAttach(file.URL()))
}

func escapeAttach(value string) string {
	return strings.Replace(escape(value), `]`, `\]`, -1)
}

func repartition(lines []string) []string {
	parts := make([]string, 0)
	curr := ""
	for _, line := range lines {
		test := curr
		if len(test) > 0 {
			test += "\n"
		}

		test += line
		if len(test) > maxMessageLength {
			parts = append(parts, curr)
			curr = ""
		} else {
			curr = test
		}
	}

	if len(curr) > 0 {
		parts = append(parts, curr)
	}

	if len(parts) == 0 {
		parts = append(parts, "")
	}

	return parts
}

var escaper = strings.NewReplacer(
	`\r`, ``,
	`\`, `\\`,
	`[`, `\[`,
	`*`, `\*`,
	`_`, `\_`)

func escape(text string) string {
	return fixUnderscores(escaper.Replace(text))
}

var underscoreRegexp = regexp.MustCompile(`([a-zA-Zа-яА-Я0-9])\\_([a-zA-Zа-яА-Я0-9])`)

func fixUnderscores(text string) string {
	text = underscoreRegexp.ReplaceAllString(text, "$1_$2")
	return text
}

func attr(attrs []html.Attribute, name string) (string, error) {
	for _, a := range attrs {
		if a.Key == name {
			return a.Val, nil
		}
	}

	return "", errors.New("no '" + name + "' attribute")
}
