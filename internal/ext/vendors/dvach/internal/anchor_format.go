package internal

import (
	"fmt"
	"strings"

	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"golang.org/x/net/html"
)

type AnchorFormat struct {
	Board string
}

func (f AnchorFormat) Format(text string, attrs []html.Attribute) string {
	if dataNum := tghtml.Get(attrs, "data-num"); dataNum != "" {
		return fmt.Sprintf(`#%s%s`, strings.ToUpper(f.Board), dataNum)
	} else {
		return tghtml.DefaultAnchorFormat.Format(text, attrs)
	}
}
