package event

import (
	"context"
	"time"

	null "gopkg.in/guregu/null.v3"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Log struct {
	Time      time.Time   `gorm:"not null;index"`
	Type      string      `gorm:"not null;index:idx_event"`
	ChatID    telegram.ID `gorm:"not null;index:idx_event"`
	MessageID telegram.ID `gorm:"not null;index:idx_event"`
	UserID    telegram.ID `gorm:"not null;index"`
	Subreddit null.String
	ThingID   null.String
}

func (l *Log) TableName() string {
	return "event"
}

type Storage interface {
	SaveEvent(ctx context.Context, row *Log) error
	DeleteEvents(ctx context.Context, chatID, userID, messageID telegram.ID, types ...string) error
	CountEvents(ctx context.Context, chatID, messageID telegram.ID, types ...string) (map[string]int64, error)
}
