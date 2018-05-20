package text

import (
	"fmt"
	"html"
	"strings"

	"github.com/jfk9w-go/misc"
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

func (b *htmlBuilder) write(value string, args ...interface{}) {
	value = fmt.Sprintf(value, args...)
	b.sb.WriteString(value)
	b.size += misc.RuneLength(value)
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
		startTag := misc.SliceRunes(*b.startTag, 1, -1)
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
	length := misc.RuneLength(text)
	if length <= capacity {
		return 0
	}

	newLine := misc.FindLastRune(text, '\n', 0, capacity)
	if newLine > 0 {
		return length - newLine - 1
	}

	space := misc.FindLastRune(text, ' ', 0, capacity)
	if space > 0 {
		return length - space - 1
	}

	return length - capacity
}

func (b *htmlBuilder) writeText(text string) {
	text = html.EscapeString(text)
	length := misc.RuneLength(text)
	offset := 0
	for left := b.fillChunk(text); left > 0; left = b.fillChunk(misc.SliceRunes(text, offset, 0)) {
		start := offset
		offset = length - left
		part := misc.SliceRunes(text, start, offset)
		b.write(strings.Trim(part, " \n"))
		b.newChunk()
	}

	b.write(misc.SliceRunes(text, offset, 0))
}

func (b *htmlBuilder) writeLink(link string) {
	if misc.RuneLength(link) > b.chunkSize-b.size {
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
