package text

import (
	"html"
	"strings"

	"github.com/jfk9w-go/gox/utf8x"
)

type htmlBuilder struct {
	chunkSize int

	sb   *strings.Builder
	size int

	startTag *string
	tagDepth int

	chunks []string
}

func newHtmlBuilder(chunkSize int) *htmlBuilder {
	return &htmlBuilder{chunkSize, &strings.Builder{}, 0, nil, 0, make([]string, 0)}
}

func (b *htmlBuilder) write(value string) {
	b.sb.WriteString(value)
	b.size += utf8x.Size(value)
}

func (b *htmlBuilder) writeStartTag(tag string) {
	if b.startTag == nil {
		b.startTag = &tag
		b.write(tag)
	}

	b.tagDepth++
}

func (b *htmlBuilder) writeEndTag() {
	b.tagDepth--

	if b.tagDepth == 0 {
		startTag := utf8x.Slice(*b.startTag, 1, -1)
		tag := strings.Fields(startTag)[0]
		b.write("</" + tag + ">")
		b.startTag = nil
	}
}

func (b *htmlBuilder) newChunk() {
	tag := b.startTag
	if tag != nil {
		b.writeEndTag()
	}

	b.chunks = append(b.chunks, b.sb.String())
	b.sb.Reset()
	b.size = 0

	if tag != nil {
		b.writeStartTag(*tag)
	}
}

func (b *htmlBuilder) fillChunk(text string) int {
	capacity := b.chunkSize - b.size
	length := utf8x.Size(text)
	if length <= capacity {
		return 0
	}

	newLine := utf8x.LastIndexOf(text, '\n', 0, capacity)
	if newLine > 0 {
		return length - newLine - 1
	}

	space := utf8x.LastIndexOf(text, ' ', 0, capacity)
	if space > 0 {
		return length - space - 1
	}

	return length - capacity
}

func (b *htmlBuilder) writeText(text string) {
	text = html.EscapeString(text)
	length := utf8x.Size(text)
	offset := 0
	for left := b.fillChunk(text); left > 0; left = b.fillChunk(utf8x.Slice(text, offset, 0)) {
		start := offset
		offset = length - left
		part := utf8x.Slice(text, start, offset)
		if b.size == 0 {
			part = strings.TrimLeft(part, " \n")
		}

		b.write(part)
		b.newChunk()
	}

	b.write(utf8x.Slice(text, offset, 0))
}

func (b *htmlBuilder) writeLink(link string) {
	if utf8x.Size(link) > b.chunkSize-b.size {
		b.newChunk()
	}

	b.write(link)
}

func (b *htmlBuilder) get() []string {
	if b.size == 0 {
		return b.chunks
	}

	return append(b.chunks, b.sb.String())
}
