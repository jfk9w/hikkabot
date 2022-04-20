package reddit

import (
	"context"
	"time"

	"hikkabot/3rdparty/reddit"
	"hikkabot/feed"

	"github.com/jfk9w-go/flu"
)

type Score struct {
	First          *time.Time
	LikedThings    int
	DislikedThings int
	Likes          int
	Dislikes       int
}

type StorageTx interface {
	Score(feedID feed.ID, thingIDs []string) (*Score, error)
	GetPercentile(subreddit string, top float64) (int, error)
	GetFreshThingIDs(ids flu.Set[string]) (flu.Set[string], error)
	DeleteStaleThings(until time.Time) (int64, error)
}

type StorageInterface interface {
	feed.Storage
	feed.EventStorage
	SaveThings(ctx context.Context, things []reddit.Thing) error
	RedditTx(ctx context.Context, body func(tx StorageTx) error) error
}
