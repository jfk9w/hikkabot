package subreddit

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"

	"hikkabot/3rdparty/reddit"
	"hikkabot/3rdparty/viddit"
	"hikkabot/core/media"
	"hikkabot/util"
)

var (
	clickCommandKey   = "sr_c"
	likeCommandKey    = "sr_l"
	dislikeCommandKey = "src_dl"
)

type Context struct {
	flu.Clock
	Metrics        me3x.Registry
	Storage        Storage
	MediaManager   *media.Manager
	RedditClient   *reddit.Client
	VidditClient   *viddit.Client
	TelegramClient telegram.Client
}

type Pacing struct {
	Stable     time.Duration
	Base, Min  float64
	Multiplier float64
	MinMembers int64
	MaxBatch   int
}

type Score struct {
	First          *time.Time
	LikedThings    int
	DislikedThings int
	Likes          int
	Dislikes       int
}

type Data struct {
	Subreddit     string         `json:"subreddit"`
	SentIDs       util.StringSet `json:"sent_ids,omitempty"`
	LastCleanSecs int64          `json:"last_clean,omitempty"`
	Layout        Layout         `json:"layout,omitempty"`
}

func (d *Data) Labels() me3x.Labels {
	return me3x.Labels{}
}

type Storage interface {
	SaveThings(ctx context.Context, things []reddit.Thing) error
	DeleteStaleThings(ctx context.Context, until time.Time) (int64, error)
	GetPercentile(ctx context.Context, subreddit string, top float64) (int, error)
	GetFreshThingIDs(ctx context.Context, ids util.StringSet) (util.StringSet, error)
	Score(ctx context.Context, chatID telegram.ID, thingIDs []string) (*Score, error)
}
