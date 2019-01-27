package service

import (
	"encoding/json"
	"fmt"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Feed struct {
	ID           string
	SecondaryID  string
	ChatID       telegram.ID
	ServiceID    string
	Name         string
	OptionsBytes []byte
	Offset       int64
}

func (f *Feed) String() string {
	return fmt.Sprintf("[ %s | %s | %s ]",
		f.ChatID, f.ServiceID, string(f.OptionsBytes))
}

func (f *Feed) OptionsFunc() OptionsFunc {
	return func(value interface{}) error {
		return json.Unmarshal(f.OptionsBytes, value)
	}
}
