package pref

import (
	"context"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	null "gopkg.in/guregu/null.v3"
)

type Interaction struct {
	ChatID    telegram.ID `gorm:"primaryKey"`
	MessageID telegram.ID `gorm:"primaryKey"`
	UserID    telegram.ID `gorm:"primaryKey"`
	Time      time.Time   `gorm:"not null"`
	Like      bool        `gorm:"not null"`
	Subreddit null.String
	ThingID   null.String
}

func (i *Interaction) TableName() string {
	return "pref"
}

type Storage interface {
	SaveInteraction(ctx context.Context, interaction *Interaction) (likes int64, dislikes int64, err error)
}
