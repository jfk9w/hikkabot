package format

import telegram "github.com/jfk9w-go/telegram-bot-api"

type Paged interface {
	Pages() []string
	ParseMode() telegram.ParseMode
}
