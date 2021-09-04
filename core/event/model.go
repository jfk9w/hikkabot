package event

import (
	"context"
	"time"

	null "gopkg.in/guregu/null.v3"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

type Log struct {
	Time      time.Time   `gorm:"not null;index"`
	Type      string      `gorm:"not null;index:event_idx"`
	ChatID    telegram.ID `gorm:"not null;index:event_idx"`
	UserID    telegram.ID `gorm:"not null;index:event_idx"`
	MessageID telegram.ID `gorm:"not null;index:event_idx"`
	Subreddit null.String
	ThingID   null.String
}

func (l *Log) TableName() string {
	return "event"
}

type Storage interface {
	SaveEvent(ctx context.Context, row *Log) error
}
