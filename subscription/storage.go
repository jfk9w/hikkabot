package subscription

import telegram "github.com/jfk9w-go/telegram-bot-api"

type Change struct {
	Offset int64
	Error  error
}

type Storage interface {
	AddItem(chatID telegram.ID, item Item) (*ItemData, bool)
	GetItem(primaryID string) (*ItemData, bool)
	GetNextItem(chatID telegram.ID) (*ItemData, bool)
	Update(id string, change Change) bool
	GetActiveChats() []telegram.ID
}
