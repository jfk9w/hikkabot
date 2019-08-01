package subscription

import (
	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Storage interface {
	AddItem(chatID telegram.ID, item Item) (string, bool)
	GetItem(primaryID string) (*itemData, bool)
	GetNextItem(chatID telegram.ID) (*itemData, bool)
	UpdateItemOffset(primaryID string, offset Offset) bool
	UpdateItemError(primaryID string, err error) (*itemData, bool)
	GetActiveChats() []telegram.ID
}
