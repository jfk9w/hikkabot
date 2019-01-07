package service

import (
	"regexp"

	. "github.com/jfk9w-go/telegram-bot-api"

	. "github.com/jfk9w-go/hikkabot/common"
)

type ServiceType string

const (
	DvachCatalogType ServiceType = "2ch/catalog"
	RedditType       ServiceType = "reddit"

	MaxPhotoSize = 10 * (2 << 20)
	MaxVideoSize = 50 * (2 << 20)
)

type SearchService interface {
	Search(input string, query *regexp.Regexp, limit int) ([]Envelope, error)
}

type SubscribeService interface {
	ServiceType() ServiceType
	Subscribe(input string, chatId ID, options string) error
	Update(offset Offset, rawOptions RawOptions, feed *Feed)
	Suspend(id string, err error)
}
