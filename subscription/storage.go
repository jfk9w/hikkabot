package subscription

import telegram "github.com/jfk9w-go/telegram-bot-api"

type Storage interface {
	AddItem(chatID telegram.ID, item Item) (*ItemData, bool)
	GetItem(primaryID string) (*ItemData, bool)
	GetNextItem(chatID telegram.ID) (*ItemData, bool)
	UpdateOffset(primaryID string, offset int64) bool
	UpdateError(primaryID string, err error) bool
	ResetError(primaryID string) bool
	GetActiveChats() []telegram.ID
}
