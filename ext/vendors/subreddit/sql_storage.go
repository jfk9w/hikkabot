package subreddit

import (
	"context"
	"time"

	gormutil "github.com/jfk9w-go/flu/gorm"

	"gorm.io/gorm"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/util"
)

type SQLStorage gorm.DB

func (s *SQLStorage) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQLStorage) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(reddit.Thing))
}

func (s *SQLStorage) SaveThings(ctx context.Context, things []reddit.Thing) error {
	return s.Unmask().WithContext(ctx).
		Clauses(gormutil.OnConflictClause(things, "primaryKey", true, nil)).
		Create(things).
		Error
}

func (s *SQLStorage) DeleteStaleThings(ctx context.Context, until time.Time) (int64, error) {
	tx := s.Unmask().WithContext(ctx).
		Where("last_seen < ?", until).
		Delete(new(reddit.Thing))
	return tx.RowsAffected, tx.Error
}

func (s *SQLStorage) GetPercentile(ctx context.Context, subreddit string, top float64) (int, error) {
	var percentile int
	return percentile, s.Unmask().WithContext(ctx).Raw( /* language=SQL */ `
		select ups from (
		  select ups, cume_dist() over (order by ups) as rank
		  from reddit
		  where subreddit = ?
		) as t 
		where t.rank > ?
		order by t.rank
		limit 1`, subreddit, 1-top).
		Scan(&percentile).
		Error
}

func (s *SQLStorage) GetFreshThingIDs(ctx context.Context, ids util.StringSet) (util.StringSet, error) {
	freshIDs := make([]string, 0)
	if err := s.Unmask().WithContext(ctx).Raw( /* language=SQL */ `
		select id from reddit
		where id in ?
		order by num_id`,
		ids.Slice()).
		Scan(&freshIDs).
		Error; err != nil {
		return nil, err
	}

	set := make(util.StringSet, len(freshIDs))
	for _, id := range freshIDs {
		set.Add(id)
	}

	return set, nil
}
