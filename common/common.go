package common

import (
	"regexp"
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/utf8x"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

func ChatTitle(chat *telegram.Chat) string {
	if chat.Type == telegram.PrivateChatType {
		return "private"
	}

	return chat.Title
}

var refTagRegexp = regexp.MustCompile(`^#?([A-Za-z]+)([0-9]+)$`)

func RefTag(ref dvach.Ref) string {
	return "#" + strings.ToUpper(ref.Board) + ref.NumString
}

func ParseRefTag(tag string) (dvach.Ref, error) {
	var groups = refTagRegexp.FindStringSubmatch(tag)
	if groups == nil {
		return dvach.Ref{}, errors.Errorf("invalid ref tag: %s", tag)
	}

	return dvach.ToRef(groups[1], groups[2])
}

var threadTagSanitizerRegexp = regexp.MustCompile("[^0-9A-Za-zА-Яа-я]+")

const maxThreadTagLength = 25

func Header(item *dvach.Item) string {
	var tag = utf8x.Head(item.Subject, maxThreadTagLength, "")
	tag = threadTagSanitizerRegexp.ReplaceAllString(tag, "_")
	var lastUnderscore = utf8x.LastIndexOf(tag, '_', 0, 0)
	if lastUnderscore > 0 && utf8x.Size(tag)-lastUnderscore < 2 {
		tag = utf8x.Slice(tag, 0, lastUnderscore)
	}

	return "#" + tag
}
