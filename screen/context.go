package screen

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

type tagType int8

const (
	messageLengthSoftLimit = 3800
	messageLengthHardLimit = 3900

	none tagType = iota
	bold
	italic
	link
	reply
)

type context struct {
	messages []string
	buf      bytes.Buffer
	length   int
	tag      tagType
	depth    int
}

func newContext() *context {
	return &context{
		messages: make([]string, 0),
		tag:      none,
	}
}

func (ctx *context) start(token html.Token) {
	if token.Data == "br" {
		ctx.buf.WriteString("\n")
		return
	}

	if ctx.depth == 0 && ctx.tag == none {
		var tag tagType

		switch token.Data {
		case "strong":
			tag = bold
			break

		case "em":
		case "span":
			tag = italic
			break

		case "a":
			if hasAttribute(token, "data-num") {
				tag = reply
			} else {
				tag = link
			}

			break

		default:
			tag = none
		}

		ctx.tag = tag
		ctx.startTag()
	}

	ctx.depth++
}

func (ctx *context) text(board string, token html.Token) {
	data := token.Data
	switch ctx.tag {
	case reply:
		ctx.write(escape("#" + strings.ToUpper(board) + data[2:]))
		return

	default:
		ctx.write(escape(data))
		return
	}
}

func escape(data string) string {
	return html.EscapeString(data)
}

func (ctx *context) write(data string) {
	if data == "" {
		return
	}

	length := ctx.length + len(data)
	if length < messageLengthSoftLimit {
		ctx.length += len(data)
		ctx.buf.WriteString(data)
		return
	}

	words := strings.Split(data, " ")
	splitWord := -1
	for i, word := range words {
		wl := len(word)
		total := ctx.length
		if total+wl < messageLengthSoftLimit {
			total += wl
			splitWord = i
		} else {
			break
		}
	}

	var current, remainder string
	if splitWord == -1 && length > messageLengthHardLimit {
		split := messageLengthHardLimit - ctx.length + 1
		current = data[:split]
		remainder = data[split:]
	} else {
		splitWord++
		current = strings.Join(words[:splitWord], " ")
		remainder = strings.Join(words[splitWord:], " ")
	}

	ctx.writeSafe(current)
	ctx.dump()
	ctx.write(remainder)
}

func (ctx *context) writeSafe(data string) {
	ctx.buf.WriteString(data)
	ctx.length += len(data)
}

func (ctx *context) end(token html.Token) {
	ctx.depth--
	if ctx.depth < 0 {
		ctx.depth = 0
		return
	}

	if ctx.depth == 0 {
		ctx.endTag()
		ctx.tag = none
	}
}

func (ctx *context) dump() {
	ctx.endTag()
	ctx.messages = append(ctx.messages, ctx.buf.String())
	ctx.buf = bytes.Buffer{}
	ctx.length = 0
	ctx.startTag()
}

func (ctx *context) startTag() {
	switch ctx.tag {
	case bold:
		ctx.writeSafe("<strong>")
		return

	case italic:
		ctx.writeSafe("<em>")
		return
	}
}

func (ctx *context) endTag() {
	switch ctx.tag {
	case bold:
		ctx.writeSafe("</strong>")
		return

	case italic:
		ctx.writeSafe("</em>")
		return
	}
}

func hasAttribute(token html.Token, key string) bool {
	for _, attr := range token.Attr {
		if attr.Key == key {
			return true
		}
	}

	return false
}
