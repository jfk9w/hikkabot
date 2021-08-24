package thread

import (
	"fmt"
	"strings"

	tghtml "github.com/jfk9w-go/telegram-bot-api/ext/html"
	"golang.org/x/net/html"
)

type anchorFormat struct {
	board string
}

func (f anchorFormat) Format(text string, attrs []html.Attribute) string {
	if dataNum := tghtml.Get(attrs, "data-num"); dataNum != "" {
		return fmt.Sprintf(`#%s%s`, strings.ToUpper(f.board), dataNum)
	} else {
		return tghtml.DefaultAnchorFormat.Format(text, attrs)
	}
}
