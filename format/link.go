package format

import (
	"fmt"

	"golang.org/x/net/html"
)

type Link struct {
	Attrs []html.Attribute
	Text  string
}

func (l *Link) Attr(key string) (string, bool) {
	for _, attr := range l.Attrs {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

type LinkPrinter interface {
	Print(*Link) string
}

var DefaultLinkPrinter LinkPrinter = defaultLinkPrinter{}

type defaultLinkPrinter struct{}

func (defaultLinkPrinter) Print(link *Link) string {
	if href, ok := link.Attr("href"); ok {
		return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(href), link.Text)
	} else {
		return ""
	}
}

func PrintHTMLLink(text, href string) string {
	return DefaultLinkPrinter.Print(&Link{[]html.Attribute{{Key: "href", Val: href}}, text})
}
