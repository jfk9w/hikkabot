package subreddit

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/util"
)

type Data struct {
	Subreddit     string         `json:"subreddit"`
	SentIDs       util.Uint64Set `json:"sent_ids,omitempty"`
	Top           float64        `json:"top"`
	LastCleanSecs int64          `json:"last_clean,omitempty"`
	MediaOnly     bool           `json:"media_only,omitempty"`
	IndexUsers    bool           `json:"index_users,omitempty"`
}

func (d *Data) Labels() metrics.Labels {
	return metrics.Labels{}.
		Add("top", d.Top)
}

type Storage interface {
	SaveThings(ctx context.Context, things []reddit.Thing) error
	DeleteStaleThings(ctx context.Context, until time.Time) (int64, error)
	GetPercentile(ctx context.Context, subreddit string, top float64) (int, error)
	GetFreshThingIDs(ctx context.Context, subreddit string, ids util.Uint64Set) (util.Uint64Set, error)
}
