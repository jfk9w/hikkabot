package format

import (
	"math"
	"strings"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"golang.org/x/exp/utf8string"
	"golang.org/x/net/html"
)

type HTMLWriter struct {
	builder     strings.Builder
	tag         *tag
	tags        SupportedTags
	link        *Link
	linkPrinter LinkPrinter
	pageSize    int
	maxPageSize int
	pages       []string
	maxPages    int
}

func NewHTML(maxPageSize, maxPages int, tags SupportedTags, linkPrinter LinkPrinter) *HTMLWriter {
	if tags == nil {
		tags = DefaultSupportedTags
	}
	if linkPrinter == nil {
		linkPrinter = DefaultLinkPrinter
	}
	return &HTMLWriter{
		builder:     strings.Builder{},
		tags:        tags,
		linkPrinter: linkPrinter,
		maxPageSize: maxPageSize,
		pages:       make([]string, 0),
		maxPages:    maxPages,
	}
}

func (w *HTMLWriter) isOutOfBounds() bool {
	return w.maxPages >= 1 && len(w.pages) > w.maxPages
}

func (w *HTMLWriter) write(text string) {
	w.builder.WriteString(text)
	w.pageSize += utf8.RuneCountInString(text)
}

func (w *HTMLWriter) writeTagStart() bool {
	if w.tag != nil {
		return w.writeUnbreakable("<" + w.tag.name + ">")
	}
	return true
}

func (w *HTMLWriter) writeTagEnd() {
	if w.tag != nil {
		w.write("</" + w.tag.name + ">")
	}
}

func (w *HTMLWriter) breakPage() bool {
	if w.isOutOfBounds() {
		return false
	}
	if w.pageSize > w.tag.startLen() {
		w.write(w.tag.end())
		w.pages = append(w.pages, w.builder.String())
		w.builder.Reset()
		w.pageSize = 0
		if w.isOutOfBounds() {
			return false
		}
		w.write(w.tag.start())
	}
	return true
}

func (w *HTMLWriter) capacity() int {
	if w.maxPageSize < 1 {
		return math.MaxInt32
	}
	capacity := w.maxPageSize - w.pageSize
	if w.tag != nil {
		capacity -= w.tag.endLen()
	}
	return capacity
}

func (w *HTMLWriter) writeBreakable(text string) bool {
	if w.isOutOfBounds() {
		return false
	}
	utf8Text := utf8string.NewString(text)
	length := utf8Text.RuneCount()
	offset := 0
	capacity := w.capacity()
	end := offset + capacity
	for end < length {
		nextOffset := end
	search:
		for i := end; i >= 0; i-- {
			switch utf8Text.At(i) {
			case '\n', ' ', '\t', '\v':
				end, nextOffset = i, i+1
				break search
			case ',', '.', ':', ';':
				end, nextOffset = i+1, i+1
				break search
			default:
				continue
			}
		}
		w.write(trim(utf8Text, offset, end))
		if !w.breakPage() {
			return false
		}
		offset = nextOffset
		capacity = w.capacity()
		end = offset + capacity
	}
	w.write(utf8Text.Slice(offset, length))
	return true
}

func trim(str *utf8string.String, start, end int) string {
	return strings.Trim(str.Slice(start, end), " \t\n\v")
}

func (w *HTMLWriter) writeUnbreakable(text string) bool {
	if w.isOutOfBounds() {
		return false
	}
	length := utf8.RuneCountInString(text)
	if length > w.capacity() {
		if !w.breakPage() {
			return false
		}
		if length > w.capacity() {
			return w.writeBreakable("BROKEN")
		} else {
			w.write(text)
		}
	} else {
		w.write(text)
	}
	return true
}

func (w *HTMLWriter) Pages() []string {
	if w.breakPage() {
		return w.pages
	} else {
		if len(w.pages) > 0 {
			return w.pages[:len(w.pages)-1]
		}
		return w.pages
	}
}

func (w *HTMLWriter) StartTag(name string, attrs []html.Attribute) *HTMLWriter {
	if w.isOutOfBounds() {
		return w
	}
	if name == "br" {
		return w.NewLine()
	}
	if name == "a" {
		if w.link == nil {
			w.link = &Link{Attrs: attrs}
		}
		return w
	}
	if w.tag != nil {
		w.tag.depth++
		return w
	}
	if name, ok := w.tags.Get(name, attrs); ok {
		w.tag = &tag{name, 0}
		w.writeTagStart()
	}
	return w
}

func (w *HTMLWriter) EndTag() *HTMLWriter {
	if w.isOutOfBounds() {
		return w
	}
	if w.link != nil {
		link := w.link
		w.link = nil
		w.writeUnbreakable(w.linkPrinter.Print(link))
		return w
	}
	if w.tag != nil {
		w.tag.depth--
		if w.tag.depth <= 0 {
			w.write(w.tag.end())
			w.tag = nil
		}
	}
	return w
}

func (w *HTMLWriter) Tag(name string) *HTMLWriter {
	w.StartTag(name, nil)
	return w
}

func (w *HTMLWriter) Text(text string) *HTMLWriter {
	text = html.EscapeString(text)
	if w.link != nil {
		w.link.Text = text
	} else {
		w.writeBreakable(text)
	}
	return w
}

func (w *HTMLWriter) NewLine() *HTMLWriter {
	if w.capacity() >= 5 {
		w.write("\n")
	} else {
		w.breakPage()
	}
	return w
}

func (w *HTMLWriter) Link(text, href string) *HTMLWriter {
	return w.StartTag("a", []html.Attribute{{Key: "href", Val: href}}).
		Text(text).
		EndTag()
}

func (w *HTMLWriter) Parse(raw string) *HTMLWriter {
	reader := strings.NewReader(raw)
	tokenizer := html.NewTokenizer(reader)
	for {
		if w.isOutOfBounds() {
			return w
		}
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		data := token.Data
		switch token.Type {
		case html.TextToken:
			w.Text(data)
		case html.StartTagToken:
			w.StartTag(data, token.Attr)
		case html.EndTagToken:
			w.EndTag()
		}
	}
	return w
}

func (w *HTMLWriter) ParseMode() telegram.ParseMode {
	return telegram.HTML
}
