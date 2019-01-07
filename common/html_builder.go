package common

import (
	"fmt"
	"strings"

	"golang.org/x/exp/utf8string"
	"golang.org/x/net/html"
)

type currentTag struct {
	tag   string
	attrs []html.Attribute
	depth int
}

type HtmlBuilder struct {
	b      *strings.Builder
	tag    *currentTag
	size   int
	csize  int
	limit  int
	chunks []string
}

func NewHtmlBuilder(csize, limit int) *HtmlBuilder {
	return &HtmlBuilder{
		b:      new(strings.Builder),
		csize:  csize,
		limit:  limit,
		chunks: make([]string, 0),
	}
}

func (b *HtmlBuilder) inactive() bool {
	return len(b.chunks) >= b.limit
}

func (b *HtmlBuilder) flush() {
	if b.inactive() {
		return
	}

	b.chunks = append(b.chunks, strings.Trim(b.b.String(), " \n\t\v"))
	b.b.Reset()
	b.size = 0
}

func (b *HtmlBuilder) startTag(tag string, attrs []html.Attribute) *HtmlBuilder {
	if b.inactive() {
		return b
	}

	if b.tag != nil {
		b.tag.depth += 1
		return b
	}

	b.tag = &currentTag{tag, attrs, 0}
	b.size += len(tag) + 2

	b.b.WriteString("<" + tag)
	for _, attr := range attrs {
		str := fmt.Sprintf(` %s="%s"`, attr.Key, attr.Val)
		b.b.WriteString(str)
		b.size += len(str)
	}

	b.b.WriteString(">")
	return b
}

func (b *HtmlBuilder) endTag(tag string) *HtmlBuilder {
	if b.inactive() || b.tag == nil {
		return b
	}

	if b.tag.depth > 0 {
		b.tag.depth--
		return b
	}

	if b.tag.tag != tag {
		return b
	}

	b.b.WriteString("</" + tag + ">")
	b.size += len(tag) + 3
	b.tag = nil

	return b
}

func (b *HtmlBuilder) Text(text string) *HtmlBuilder {
	if b.inactive() {
		return b
	}

	text = html.EscapeString(text)
	str := utf8string.NewString(text)
	free := b.csize - b.size
	if str.RuneCount() < free {
		b.b.WriteString(text)
		b.size += str.RuneCount()
		return b
	}

	end, start := prettyBreak(str, free)
	b.b.WriteString(str.Slice(0, end))

	tag := b.tag
	if tag != nil {
		b.endTag(tag.tag)
	}

	b.flush()
	if b.inactive() {
		return b
	}

	if tag != nil {
		b.startTag(tag.tag, tag.attrs)
	}

	return b.Text(str.Slice(start, str.RuneCount()))
}

func (b *HtmlBuilder) Link(href, text string) *HtmlBuilder {
	if b.inactive() {
		return b
	}

	tag := b.tag
	if tag != nil {
		b.endTag(tag.tag)
	}

	link := fmt.Sprintf(`<a href="%s">%s</a>`, href, html.EscapeString(text))
	str := utf8string.NewString(link)
	free := b.csize - b.size
	if str.RuneCount() > free {
		b.flush()
		if b.inactive() {
			return b
		}
	}

	b.b.WriteString(link)
	b.size += str.RuneCount()

	if tag != nil {
		b.startTag(tag.tag, tag.attrs)
	}

	return b
}

func (b *HtmlBuilder) Br() *HtmlBuilder {
	return b.Text("\n")
}

func (b *HtmlBuilder) B() *HtmlBuilder {
	return b.startTag("b", nil)
}

func (b *HtmlBuilder) B_() *HtmlBuilder {
	return b.endTag("b")
}

func (b *HtmlBuilder) I() *HtmlBuilder {
	return b.startTag("i", nil)
}

func (b *HtmlBuilder) I_() *HtmlBuilder {
	return b.endTag("i")
}

func (b *HtmlBuilder) Span() *HtmlBuilder {
	return b.startTag("span", nil)
}

func (b *HtmlBuilder) Span_() *HtmlBuilder {
	return b.endTag("span")
}

var tags = map[string]string{
	"strong": "b",
	"b":      "b",
	"italic": "i",
	"em":     "i",
	"i":      "i",
	"span":   "i",
}

func (b *HtmlBuilder) Parse(rawinput string) *HtmlBuilder {
	reader := strings.NewReader(rawinput)
	tokenizer := html.NewTokenizer(reader)

	var currHref *string
	for {
		if b.inactive() {
			break
		}

		token := tokenizer.Next()
		if token == html.ErrorToken {
			break
		}

		data := tokenizer.Token().Data
		attrs := tokenizer.Token().Attr
		switch token {
		case html.TextToken:
			if currHref != nil {
				if *currHref != "" {
					b.Link(*currHref, data)
				}

				continue
			}

			b.Text(data)

		case html.StartTagToken:
			if data == "br" {
				b.Br()
			}

			if data == "a" {
				if href, ok := attr(attrs, "href"); ok {
					currHref = &href
					continue
				}
			}

			if mapped, ok := tags[data]; ok {
				b.startTag(mapped, nil)
				continue
			}

		case html.EndTagToken:
			if currHref != nil {
				currHref = nil
				continue
			}

			if mapped, ok := tags[data]; ok {
				b.endTag(mapped)
			}
		}
	}

	return b
}

func (b *HtmlBuilder) Done(header, footer string) []string {
	if b.tag != nil {
		b.endTag(b.tag.tag)
	}

	if b.size > 0 {
		b.flush()
	}

	if header != "" || footer != "" {
		if len(b.chunks) == 0 {
			b.chunks = append(b.chunks, "")
		}

		if header != "" {
			b.chunks[0] = header + "\n" + b.chunks[0]
		} else {
			b.chunks[len(b.chunks)-1] += "\n" + footer
		}
	}

	return b.chunks
}

func attr(attrs []html.Attribute, key string) (string, bool) {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Val, true
		}
	}

	return "", false
}

func prettyBreak(str *utf8string.String, free int) (end, start int) {
	for i := free; i >= 0; i-- {
		switch str.At(i) {
		case '\n', ' ', '\t', '\v':
			return i, i + 1
		case ',':
			return i + 1, i + 1
		default:
			continue
		}
	}

	return 0, 0
}
