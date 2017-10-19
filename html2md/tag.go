package html2md

import (
	"fmt"
	"golang.org/x/net/html"
)

type tag struct {
	typ       string
	token     string
	recognize func(t html.Token) (string, bool)
	contents  bool
	close     bool
}

func defaultTag(typ string, token string) tag {
	return tag{
		typ:       typ,
		token:     token,
		recognize: nil,
		contents:  true,
		close:     true,
	}
}

var (
	strong    = defaultTag("strong", "*")
	italic    = defaultTag("em", "_")
	greenText = defaultTag("span", "")
	link      = defaultTag("a", "")

	spoiler = tag{
		typ:   "span",
		token: "_",
		recognize: func(t html.Token) (string, bool) {
			if val, err := attr(t.Attr, "class"); err == nil && val == "spoiler" {
				return "_", true
			}

			return "", false
		},
		contents: true,
		close:    true,
	}

	reply = tag{
		typ:   "a",
		token: "",
		recognize: func(t html.Token) (string, bool) {
			if val, err := attr(t.Attr, "class"); err == nil && val == "post-reply-link" {
				if num, err := attr(t.Attr, "data-num"); err == nil {
					return fmt.Sprintf("#P%s", num), true
				} else {
					return "", false
				}
			}

			return "", false
		},
		contents: false,
		close:    false,
	}
)

func unknown(typ string) tag {
	return tag{
		typ:       typ,
		token:     "",
		recognize: nil,
		contents:  true,
	}
}

func start(t html.Token) tag {
	if t.Type != html.StartTagToken && t.Type != html.EndTagToken {
		panic("not a start or end token")
	}

	switch t.Data {
	case "strong":
		return strong
	case "em":
		return italic
	case "span":
		if token, ok := spoiler.recognize(t); ok {
			spoiler.token = token
			return spoiler
		} else {
			return greenText
		}
	case "a":
		if token, ok := reply.recognize(t); ok {
			reply.token = token
			return reply
		} else {
			return link
		}
	}

	return unknown(t.Data)
}
