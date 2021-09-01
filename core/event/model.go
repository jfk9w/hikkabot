package event

import (
	"context"
	"time"

	null "gopkg.in/guregu/null.v3"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Log struct {
	Time      time.Time   `gorm:"not null"`
	Type      string      `gorm:"not null"`
	ChatID    telegram.ID `gorm:"not null"`
	UserID    telegram.ID `gorm:"not null"`
	MessageID telegram.ID `gorm:"not null"`
	Subreddit null.String
	ThingID   null.String
}

func (l *Log) TableName() string {
	return "event_log"
}

type Storage interface {
	IsKnownUser(ctx context.Context, userID telegram.ID) (bool, error)
	SaveEvent(ctx context.Context, row *Log) error
}
