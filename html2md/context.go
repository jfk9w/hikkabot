package html2md

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type tagType int32

var (
	errInvalidDepth = errors.New("invalid depth")

	replacer = strings.NewReplacer(
		`*`, `\*`,
		`_`, `\_`,
	)
)

const (
	messageLengthSoftLimit = 3700
	messageLengthHardLimit = 3800

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
		ctx.buf.WriteRune('\n')
		return
	}

	if ctx.depth == 0 && ctx.tag == none {
		var tag tagType

		switch token.Data {
		case "strong":
			ctx.buf.WriteRune('*')
			tag = bold
			break

		case "em":
		case "span":
			ctx.buf.WriteRune('_')
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
	}

	ctx.depth++
}

func (ctx *context) text(token html.Token) {
	data := token.Data
	switch ctx.tag {
	case reply:
		ctx.buf.WriteString("#P" + data[2:])
		return

	case link:
		ctx.buf.WriteString(escape(data))
		return

	default:
		ctx.write(escape(data))
	}
}

func escape(data string) string {
	return replacer.Replace(data)
}

func (ctx *context) write(data string) {
	if data == "" {
		return
	}

	length := ctx.length + len(data)
	var current, remainder string
	if length < messageLengthSoftLimit {
		ctx.buf.WriteString(data)
		return
	}

	words := strings.Split(data, " ")
	splitWord := -1
	for i, word := range words {
		wl := len(word)
		if ctx.length+wl < messageLengthSoftLimit {
			ctx.length++
			splitWord = i
		} else {
			break
		}
	}

	if splitWord == -1 && length > messageLengthHardLimit {
		split := messageLengthHardLimit - ctx.length + 1
		current = data[:split]
		remainder = data[split:]
	} else {
		splitWord++
		current = strings.Join(words[:splitWord], " ")
		remainder = strings.Join(words[splitWord:], " ")
	}

	ctx.buf.WriteString(current)
	ctx.dump()
	ctx.write(remainder)
}

func (ctx *context) end(token html.Token) error {
	ctx.depth--
	if ctx.depth < 0 {
		ctx.depth = 0
		return errInvalidDepth
	}

	var err error
	if ctx.depth == 0 {
		var check bool
		switch token.Data {
		case "strong":
			check = ctx.tag == bold
			break

		case "em":
		case "span":
			check = ctx.tag == italic
			break

		case "a":
			check = ctx.tag == reply || ctx.tag == link
			break

		default:
			check = ctx.tag == none
		}

		if !check {
			err = fmt.Errorf("invalid tag: %s != %d", token.Data, ctx.tag)
		}

		ctx.writeTag()
		ctx.tag = none
	}

	return err
}

func (ctx *context) dump() {
	ctx.writeTag()
	ctx.messages = append(ctx.messages, ctx.buf.String())
	ctx.buf = bytes.Buffer{}
	ctx.length = 0
	ctx.writeTag()
}

func (ctx *context) writeTag() {
	if ctx.tag == bold {
		ctx.buf.WriteRune('*')
		ctx.length++
	} else if ctx.tag == italic {
		ctx.buf.WriteRune('_')
		ctx.length++
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
