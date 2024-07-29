package html

import (
	"fmt"

	"golang.org/x/net/html"
)

type anchor struct {
	text   string
	attrs  []html.Attribute
	parent *Tag
}

type defaultAnchorFormat struct{}

func (f defaultAnchorFormat) Format(text string, attrs []html.Attribute) string {
	href := Get(attrs, "href")
	if href != "" {
		return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(href), text)
	}
	return ""
}

var DefaultAnchorFormat AnchorFormat = defaultAnchorFormat{}

func Anchor(text, href string) string {
	return DefaultAnchorFormat.Format(html.EscapeString(text), []html.Attribute{{Key: "href", Val: href}})
}
