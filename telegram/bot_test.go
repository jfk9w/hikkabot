package telegram

import (
	"testing"
)

func TestGetMe(t *testing.T) {
	bot := NewBotAPI(nil, "")
	bot.Start(&GetUpdatesRequest{
		Timeout: 2,
	})

	bot.Stop(false)
}