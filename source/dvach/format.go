package dvach

import (
	"fmt"
	"strings"

	"github.com/jfk9w/hikkabot/format"
	"golang.org/x/net/html"
)

var DefaultSupportedTags format.SupportedTags = defaultSupportedTags{}

type defaultSupportedTags struct{}

func (defaultSupportedTags) Get(tag string, attrs []html.Attribute) (string, bool) {
	if tag == "span" {
		return "i", true
	} else {
		return format.DefaultSupportedTags.Get(tag, attrs)
	}
}

type Board string

func (b Board) Print(link *format.Link) string {
	if replyTo, ok := link.Attr("data-num"); ok {
		return fmt.Sprintf(`#%s%s`, strings.ToUpper(string(b)), replyTo)
	} else {
		return format.DefaultLinkPrinter.Print(link)
	}
}
