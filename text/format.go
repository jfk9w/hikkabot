package text

import (
	"fmt"
	"regexp"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/misc"
)

type Hashtag = string

func Format(board dvach.Board, numstr string) string {
	return fmt.Sprintf("#%s%s", board, numstr)
}

func FormatRef(ref dvach.Ref) Hashtag {
	return Format(ref.Board, ref.NumString)
}

var hashtagScreen = regexp.MustCompile("[^0-9A-Za-zА-Яа-я]+")

func FormatSubject(subject string) Hashtag {
	str := "#" + hashtagScreen.ReplaceAllString(subject, "_")
	runes := []rune(str)
	return string(runes[:misc.MinInt(len(runes), 27)])
}
