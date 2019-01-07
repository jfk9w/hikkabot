package service

import (
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
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

func (s *Subscription) String() string {
	return fmt.Sprintf("s (%s %s %s %s) for %s",
		s.SecondaryID, s.Type, s.Name, string(s.Options), s.ChatID.StringValue())
}
