package html

import (
	"fmt"

	"golang.org/x/net/html"
)

var (
	EscapeString   = html.EscapeString
	UnescapeString = html.UnescapeString
)

func B(text string) string {
	return "<b>" + text + "</b>"
}

func Link(href string, text string) string {
	return fmt.Sprintf(`<a href="%s">%s</a>`, href, EscapeString(text))
}
