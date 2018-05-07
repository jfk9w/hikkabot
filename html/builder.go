package html

import (
	"strings"

	"github.com/jfk9w-go/misc"
)

type Builder struct {
	chunkSize int
	chunks    []string

	sb   *strings.Builder
	size int

	startTag *string
	tagDepth int
}

func NewBuilder(chunkSize int) *Builder {
	return &Builder{chunkSize, make([]string, 0), &strings.Builder{}, 0, nil, 0}
}

func (b *Builder) writeText(text string) {
	b.sb.WriteString(text)
	b.size += len(text)
}

func (b *Builder) left(text string) int {
	capacity := b.chunkSize - b.size
	size := len(text)
	if size <= capacity {
		return 0
	}

	newLine := strings.LastIndex(text[:capacity], "\n")
	if newLine > 0 {
		return size - newLine - 1
	}

	space := strings.LastIndex(text[:capacity], " ")
	if space > 0 {
		return size - space - 1
	}

	return size - capacity
}

func (b *Builder) writeStartTag(tag string) {
	b.startTag = &tag
	b.sb.WriteString(tag)
	b.size += len(tag)
}

func (b *Builder) writeEndTag() {
	tag := strings.Split((*b.startTag)[1:len(*b.startTag)-1], " ")[0]
	b.sb.WriteString("</")
	b.sb.WriteString(tag)
	b.sb.WriteRune('>')
	b.size += len(tag) + 3
	b.startTag = nil
}

func (b *Builder) checkRoom(room int) int {
	return misc.MaxInt(0, b.size+room-b.chunkSize)
}

func (b *Builder) makeRoom() {
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

func (b *Builder) WriteNewLine() {
	b.sb.WriteRune('\n')
	b.size++
}

func (b *Builder) WriteStartTag(tag string) {
	if b.tagDepth == 0 {
		b.writeStartTag(tag)
	}
	b.tagDepth++
}

func (b *Builder) WriteEndTag() {
	b.tagDepth--
	if b.tagDepth == 0 {
		b.writeEndTag()
	}
}

func (b *Builder) WriteText(text string) {
	size := len(text)
	offset := 0
	for left := b.left(text); left > 0; left = b.left(text[offset:]) {
		start := offset
		offset = size - left
		part := text[start:offset]
		b.writeText(strings.Trim(part, " \n"))
		b.makeRoom()
	}

	b.writeText(text[offset:])
}

func (b *Builder) WriteMark() {
	b.writeText("#T\n")
}

func (b *Builder) WriteHeader(num string, hash string) {
	b.writeText(hash + "\n#" + num + "\n-------\n")
}

func (b *Builder) WriteHashtag(tag string) {
	b.writeText("#" + tag + " ")
}

func (b *Builder) WriteLink(link string) {
	if b.checkRoom(len(link)) != 0 {
		b.makeRoom()
	}

	b.writeText(link)
}

func (b *Builder) Done() []string {
	if b.size == 0 {
		return b.chunks
	}

	return append(b.chunks, b.sb.String())
}
