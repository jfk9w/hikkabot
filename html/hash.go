package html

import (
	"regexp"

	"github.com/jfk9w-go/misc"
)

var symbolsRegex = regexp.MustCompile("[^0-9A-Za-zА-Яа-я]+")

func Hash(value string) string {
	hash := "#" + symbolsRegex.ReplaceAllString(value, "_")
	return string([]rune(hash)[:misc.MinInt(len(hash), 31)])
}
