package text

import (
	"fmt"
	"regexp"

	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/misc"
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
	str = misc.FirstRunes(str, 25, "")
	last := misc.FindLastRune(str, '_', 0, 0)
	if last > 0 && misc.RuneLength(str)-last < 2 {
		str = misc.SliceRunes(str, 0, last)
	}

	runes := []rune(str)
	return string(runes[:misc.MinInt(len(runes), 27)])
}
