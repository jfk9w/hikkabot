package html

import (
	"html"
	"strings"

	"github.com/jfk9w-go/hikkabot/common/gox/utf8x"
)

type Output struct {
	builder     *strings.Builder
	chunks      []string
	maxChunks   int
	chunkSize   int
	currentSize int
	hasContent  bool
	currentTag  *string
	tagDepth    int
}

func NewOutput(maxChunks int, chunkSize int) *Output {
	return &Output{
		builder:     new(strings.Builder),
		chunks:      make([]string, 0),
		maxChunks:   maxChunks,
		chunkSize:   chunkSize,
		currentSize: 0,
		hasContent:  false,
		currentTag:  nil,
		tagDepth:    0,
	}
}

func (output *Output) append(text string) *Output {
	output.builder.WriteString(text)
	output.currentSize += utf8x.Size(text)
	return output
}

func (output *Output) appendContent(text string) *Output {
	output.hasContent = true
	return output.append(text)
}

func (output *Output) AppendStartTag(tag string) *Output {
	if output.tagDepth == 0 {
		output.append("<" + tag + ">")
		output.currentTag = &tag
	}

	output.tagDepth += 1
	return output
}

func (output *Output) AppendEndTag() *Output {
	if output.tagDepth > 0 {
		output.tagDepth -= 1
		if output.tagDepth == 0 && output.currentTag != nil {
			output.append("</" + *output.currentTag + ">")
			output.currentTag = nil
		}
	}

	return output
}

func (output *Output) Flush() []string {
	if !output.hasContent {
		return output.chunks
	}

	if output.currentTag != nil {
		output.AppendEndTag()
	}

	output.chunks = append(output.chunks,
		strings.Trim(output.builder.String(), " \n"))

	output.builder.Reset()
	output.currentSize = 0
	output.hasContent = false

	if output.currentTag != nil {
		output.AppendStartTag(*output.currentTag)
	}

	return output.chunks
}

func (output *Output) Chunks() []string {
	return output.chunks
}

func (output *Output) flushAndEnd() bool {
	output.Flush()
	return output.End()
}

func (output *Output) End() bool {
	return output.maxChunks > 0 && len(output.chunks) >= output.maxChunks
}

func (output *Output) Capacity() int {
	return output.chunkSize - output.currentSize
}

func (output *Output) Fits(text string) bool {
	return output.chunkSize < 0 || utf8x.Size(text) <= output.Capacity()
}

func (output *Output) Break(text string) int {
	if output.Fits(text) {
		return utf8x.Size(text)
	}

	var capacity = output.Capacity()

	var newLine = utf8x.LastIndexOf(text, '\n', 0, capacity)
	if newLine > 0 {
		return newLine
	}

	var space = utf8x.LastIndexOf(text, ' ', 0, capacity)
	if space > 0 {
		return space
	}

	return capacity
}

func (output *Output) AppendText(text string) *Output {
	text = html.EscapeString(text)
	for !output.Fits(text) {
		var split = output.Break(text)
		output.appendContent(utf8x.Slice(text, 0, split))
		text = utf8x.Slice(text, split, 0)
		if output.flushAndEnd() {
			return output
		}
	}

	return output.appendContent(text)
}

func (output *Output) AppendLink(link string) *Output {
	if !output.Fits(link) && output.flushAndEnd() {
		return output
	}

	return output.appendContent(link)
}
