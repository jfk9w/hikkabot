package util

import (
	"html"
	"regexp"
	"strings"

	"golang.org/x/exp/utf8string"
)

var (
	tagRegexp  = regexp.MustCompile(`<.*?>`)
	junkRegexp = regexp.MustCompile(`(?i)[^\wа-яё_]`)
)

func Hashtag(str string) string {
	str = html.UnescapeString(str)
	str = tagRegexp.ReplaceAllString(str, "")
	fields := strings.Fields(str)
	for i, field := range fields {
		fields[i] = strings.Title(junkRegexp.ReplaceAllString(field, ""))
	}
	str = strings.Join(fields, "")
	tag := utf8string.NewString(str)
	if tag.RuneCount() > 25 {
		return "#" + tag.Slice(0, 25)
	}
	return "#" + tag.String()
}
