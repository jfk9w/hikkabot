package keeper

import (
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/telegram"
)

const (
	delimChat   = ":"
	delimThread = "/"
)

func refs2key(chat telegram.ChatRef, thread dvach.Ref) string {
	return chat.String() + delimChat + thread.Board + delimThread + thread.NumString
}

func key2refs(key string) (chat telegram.ChatRef, thread dvach.Ref) {
	tokens := strings.Split(key, delimChat)
	chat, _ = telegram.ParseChatRef(tokens[0])
	tokens = strings.Split(tokens[1], delimThread)
	thread, _ = dvach.ToRef(tokens[0], tokens[1])
	return
}
