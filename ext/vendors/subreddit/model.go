package subreddit

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu/metrics"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/util"
)

var (
	clickCommandKey   = "sr_c"
	likeCommandKey    = "sr_l"
	dislikeCommandKey = "src_dl"
)

type Data struct {
	Subreddit     string         `json:"subreddit"`
	SentIDs       util.StringSet `json:"sent_ids,omitempty"`
	Top           float64        `json:"top"`
	LastCleanSecs int64          `json:"last_clean,omitempty"`
	Layout        Layout         `json:"layout,omitempty"`
}

func (d *Data) Labels() metrics.Labels {
	return metrics.Labels{}.Add("top", d.Top)
}

type Storage interface {
	SaveThings(ctx context.Context, things []reddit.Thing) error
	DeleteStaleThings(ctx context.Context, until time.Time) (int64, error)
	GetPercentile(ctx context.Context, subreddit string, top float64) (int, error)
	GetFreshThingIDs(ctx context.Context, ids util.StringSet) (util.StringSet, error)
	CountUniqueEvents(ctx context.Context, chatID telegram.ID, subreddit string, since time.Time) (map[string]int64, error)
}
