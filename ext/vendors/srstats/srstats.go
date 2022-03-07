package srstats

import (
	"context"
	"hikkabot/core/feed"
	"time"

	"github.com/jfk9w-go/telegram-bot-api"

	"hikkabot/3rdparty/srstats"
)

type Telegram telegram.Client

type Feeds interface {
	Get(ctx context.Context, header *feed.Header) (*feed.Subscription, error)
}

type Events interface {
	CountChatLikesBySubreddit(ctx context.Context, chatID telegram.ID, since time.Time) (map[string]int64, error)
}

type Stats interface {
	GetSuggestions(ctx context.Context, subreddits map[string]float64) (srstats.Suggestions, error)
}
