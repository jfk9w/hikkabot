package media

import telegram "github.com/jfk9w-go/telegram-bot-api"

const (
	MaxPhotoSize int64 = 10 * (2 << 20)
	MaxVideoSize int64 = 50 * (2 << 20)
	MinMediaSize int64 = 10 << 10
)

type TelegramType = telegram.MediaType

var unknownType TelegramType

type Type = string

var (
	minMediaSize int64 = 10 << 10
	maxMediaSize       = map[TelegramType]int64{
		telegram.Photo: MaxPhotoSize,
		telegram.Video: MaxVideoSize,
	}
)
