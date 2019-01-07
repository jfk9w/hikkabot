package common

import (
	"fmt"
	"html"
	"strings"
)

func B(text string) string {
	return "<b>" + text + "</b>"
}

func Link(href string, text string) string {
	return fmt.Sprintf(`<a href="%s">%s</a>`, href, html.EscapeString(text))
}

func DvachTag(boardId string, num string) string {
	return fmt.Sprintf("#%s%s", strings.ToUpper(boardId), num)
}
