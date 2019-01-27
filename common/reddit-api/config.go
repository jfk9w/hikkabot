package red

import (
	. "github.com/jfk9w-go/hikkabot/common/gox/jsonx"
	"github.com/jfk9w-go/hikkabot/common/httpx"
)

type Config struct {
	HTTP                *httpx.Config `json:"http"`
	ClientID            string        `json:"client_id"`
	ClientSecret        string        `json:"client_secret"`
	Username            string        `json:"username"`
	Password            string        `json:"password"`
	UserAgent           string        `json:"user_agent"`
	RefreshTokenTimeout Duration      `json:"refresh_token_timeout"`
}
