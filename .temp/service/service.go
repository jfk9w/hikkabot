package service

import telegram "github.com/jfk9w-go/telegram-bot-api"

const (
	MaxPhotoSize = 10 * (2 << 20)
	MaxVideoSize = 50 * (2 << 20)
)

type OptionsFunc func(interface{}) error

type Service interface {
	ID() string
	Subscribe(input string, chat *EnrichedChat, args string) error
	Update(prevOffset int64, optionsFunc OptionsFunc, updatePipe *UpdatePipe)
}

type Storage interface {
	ActiveSubscribers() []telegram.ID
	InsertFeed(*Feed) bool
	NextFeed(telegram.ID) *Feed
	UpdateFeedOffset(string, int64) bool
	GetFeed(string) *Feed
	ResumeFeed(string) bool
	SuspendFeed(string, error) bool
}