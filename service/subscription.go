package service

import (
	"github.com/jfk9w-go/telegram-bot-api"
)

type Subscription struct {
	ID          string
	SecondaryID string
	ChatID      telegram.ID
	Type        ServiceType
	Name        string
	Options     RawOptions
	Offset      Offset
}
