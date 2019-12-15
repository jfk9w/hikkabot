package format

import telegram "github.com/jfk9w-go/telegram-bot-api"

type Formatter interface {
	Format() Text
}

type Pages = []string

type Text struct {
	Pages     Pages
	ParseMode telegram.ParseMode
}
