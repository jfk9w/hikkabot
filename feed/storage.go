package feed

import telegram "github.com/jfk9w-go/telegram-bot-api"

type Change struct {
	Offset int64
	Error  error
}

type Storage interface {
	Create(chatID telegram.ID, item Item) (*ItemData, bool)
	Get(primaryID string) (*ItemData, bool)
	Advance(chatID telegram.ID) (*ItemData, bool)
	Update(id string, change Change) bool
	Active() []telegram.ID
}
