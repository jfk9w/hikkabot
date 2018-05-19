package text

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	spanRegex   = regexp.MustCompile(`<span.*?>`)
	tagReplacer = strings.NewReplacer(
		"<br>", "\n",
		"<strong>", "<b>",
		"</strong>", "</b>",
		"<em>", "<i>",
		"</em>", "</i>",
		"</span>", "</i>",
	)
)

func prepare(text string) *html.Tokenizer {
	text = tagReplacer.Replace(text)
	text = spanRegex.ReplaceAllString(text, "<i>")
	reader := strings.NewReader(text)
	return html.NewTokenizer(reader)
}
