package frontend

import "github.com/jfk9w-go/telegram"

type Config struct {
	TempStorage string            `json:"temp_storage"`
	Superusers  []telegram.ChatID `json:"superusers"`
}
