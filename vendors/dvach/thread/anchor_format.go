package thread

import (
	"fmt"
	"strings"

	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
)

type anchorFormat struct {
	board string
}

func (f anchorFormat) Format(text string, attrs richtext.HTMLAttributes) string {
	if dataNum := attrs.Get("data-num"); dataNum != "" {
		return fmt.Sprintf(`#%s%s`, strings.ToUpper(f.board), dataNum)
	} else {
		return richtext.DefaultHTMLAnchorFormat.Format(text, attrs)
	}
}
