package html

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

type Builder struct {
	b      *strings.Builder
	tag    *currentTag
	size   int
	csize  int
	limit  int
	chunks []string
}

func NewBuilder(csize, limit int) *Builder {
	return &Builder{
		b:      new(strings.Builder),
		csize:  csize,
		limit:  limit,
		chunks: make([]string, 0),
	}
}

func (b *Builder) inactive() bool {
	if b.limit <= 0 {
		return false
	}

	return len(b.chunks) >= b.limit
}

func (b *Builder) flush() {
	if b.inactive() {
		return
	}

	b.chunks = append(b.chunks, strings.Trim(b.b.String(), " \n\t\v"))
	b.b.Reset()
	b.size = 0
}

func (b *Builder) startTag(tag string, attrs []html.Attribute) *Builder {
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

func (b *Builder) endTag(tag string) *Builder {
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

func (b *Builder) Text(text string) *Builder {
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

func (b *Builder) Link(href, text string) *Builder {
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

func (b *Builder) Br() *Builder {
	return b.Text("\n")
}

func (b *Builder) B() *Builder {
	return b.startTag("b", nil)
}

func (b *Builder) EndB() *Builder {
	return b.endTag("b")
}

func (b *Builder) I() *Builder {
	return b.startTag("i", nil)
}

func (b *Builder) EndI() *Builder {
	return b.endTag("i")
}

func (b *Builder) Span() *Builder {
	return b.startTag("span", nil)
}

func (b *Builder) EndSpan() *Builder {
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

type Parser interface {
	OnTextToken(data string) bool
	OnStartTagToken()
}

func (b *Builder) Parse(rawinput string) *Builder {
	reader := strings.NewReader(rawinput)
	tokenizer := html.NewTokenizer(reader)

	var currHref *string
	for {
		if b.inactive() {
			break
		}

		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		data := token.Data
		attrs := token.Attr
		switch tokenType {
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

func (b *Builder) Build() []string {
	if b.tag != nil {
		b.endTag(b.tag.tag)
	}

	if b.size > 0 {
		b.flush()
	}

	if len(b.chunks) == 0 {
		b.chunks = append(b.chunks, "")
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
	limit := free
	if limit > str.RuneCount() {
		limit = str.RuneCount()
	}

	for i := limit - 1; i >= 0; i-- {
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
