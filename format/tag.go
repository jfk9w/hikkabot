package format

import "golang.org/x/net/html"

type tag struct {
	name  string
	depth int
}

func (t *tag) start() string {
	if t == nil {
		return ""
	} else {
		return "<" + t.name + ">"
	}
}

func (t *tag) startLen() int {
	if t == nil {
		return 0
	} else {
		return 2 + len(t.name)
	}
}

func (t *tag) end() string {
	if t == nil {
		return ""
	} else {
		return "</" + t.name + ">"
	}
}

func (t *tag) endLen() int {
	if t == nil {
		return 0
	} else {
		return 3 + len(t.name)
	}
}

type SupportedTags interface {
	Get(string, []html.Attribute) (string, bool)
}

type defaultSupportedTags map[string]string

func (d defaultSupportedTags) Get(tag string, attrs []html.Attribute) (string, bool) {
	tag, ok := d[tag]
	return tag, ok
}

var DefaultSupportedTags SupportedTags = defaultSupportedTags{
	"strong": "b",
	"b":      "b",
	"italic": "i",
	"em":     "i",
	"i":      "i",
	"code":   "code",
	"pre":    "pre",
}
