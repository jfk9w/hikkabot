package backend

import "github.com/jfk9w-go/telegram"

func toKey(chat telegram.ChatRef) string {
	return chat.String()
}

func fromKey(key string) telegram.ChatRef {
	return telegram.NewChatRef(key)
}
