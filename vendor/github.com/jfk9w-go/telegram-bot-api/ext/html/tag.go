package html

import "golang.org/x/net/html"

type PlainTagConverter map[string]Tag

func (c PlainTagConverter) Get(name string, _ []html.Attribute) (Tag, bool) {
	tag, ok := c[name]
	return tag, ok
}

var DefaultTagConverter = PlainTagConverter{
	"strong": Bold,
	"b":      Bold,
	"italic": Italic,
	"em":     Italic,
	"i":      Italic,
	"code":   Code,
	"pre":    Pre,
}
