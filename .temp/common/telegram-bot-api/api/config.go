package api

import "github.com/jfk9w-go/hikkabot/common/httpx"

type Config struct {
	Token   string              `json:"token"`
	Aliases map[Username]ChatID `json:"aliases"`
	Http    *httpx.Config       `json:"http"`
}
