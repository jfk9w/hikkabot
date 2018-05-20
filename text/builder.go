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
	b.size += len([]rune(value))
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
		tag := strings.Split((*b.startTag)[1:len(*b.startTag)-1], " ")[0]
		b.write("</" + tag + ">")
		b.startTag = nil
	}
}

func (b *htmlBuilder) checkRoom(room int) int {
	return misc.MaxInt(0, b.size+room-b.chunkSize)
}

func (b *htmlBuilder) makeRoom() {
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

func (b *htmlBuilder) split(text string) int {
	capacity := b.chunkSize - b.size
	size := misc.RuneLength(text)
	if size <= capacity {
		return 0
	}

	newLine := misc.FindLastRune(text, '\n', 0, capacity)
	if newLine > 0 {
		return size - newLine - 1
	}

	space := misc.FindLastRune(text, ' ', 0, capacity)
	if space > 0 {
		return size - space - 1
	}

	return size - capacity
}

func (b *htmlBuilder) writeText(text string) {
	text = html.EscapeString(text)
	size := misc.RuneLength(text)
	offset := 0
	for left := b.split(text); left > 0; left = b.split(misc.SliceRunes(text, offset, 0)) {
		start := offset
		offset = size - left
		part := misc.SliceRunes(text, start, offset)
		b.write(strings.Trim(part, " \n"))
		b.makeRoom()
	}

	b.write(misc.SliceRunes(text, offset, 0))
}

func (b *htmlBuilder) writeLink(link string) {
	if b.checkRoom(len(link)) != 0 {
		b.makeRoom()
	}

	b.write(link)
}

func (b *htmlBuilder) get() []string {
	if b.size == 0 {
		return b.chunks
	}

	return append(b.chunks, b.sb.String())
}
