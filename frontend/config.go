package frontend

import "github.com/jfk9w-go/hikkabot/common/telegram-bot-api"

type Config struct {
	TempStorage string            `json:"temp_storage"`
	Superusers  []telegram.ChatID `json:"superusers"`
}
