package content

import "github.com/jfk9w-go/telegram"

var paddedMaxMessageSize = telegram.MaxMessageSize * 4 / 5

func FormatChatTitle(chat *telegram.Chat) string {
	if chat.Type == telegram.PrivateChatType {
		return "private"
	}

	return chat.Title
}
