package text

import (
	"fmt"
	"regexp"

	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/mathx"
	"github.com/jfk9w-go/gox/utf8x"
)

type Hashtag = string

func Format(board dvach.Board, numstr string) string {
	return fmt.Sprintf("#%s%s", strings.ToUpper(board), numstr)
}

func FormatRef(ref dvach.Ref) Hashtag {
	return Format(ref.Board, ref.NumString)
}

var hashtagScreen = regexp.MustCompile("[^0-9A-Za-zА-Яа-я]+")

func FormatSubject(subject string) Hashtag {
	str := "#" + hashtagScreen.ReplaceAllString(subject, "_")
	str = utf8x.Head(str, 25, "")
	last := utf8x.LastIndexOf(str, '_', 0, 0)
	if last > 0 && utf8x.Size(str)-last < 2 {
		str = utf8x.Slice(str, 0, last)
	}

	runes := []rune(str)
	return string(runes[:mathx.MinInt(len(runes), 27)])
}
