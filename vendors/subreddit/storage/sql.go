package storage

import (
	"context"
	"time"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/util"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"gorm.io/gorm"
)

type SQL gorm.DB

func (s *SQL) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQL) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(reddit.ThingData))
}

func (s *SQL) SaveThing(ctx context.Context, thing *reddit.ThingData) error {
	return s.Unmask().WithContext(ctx).
		Clauses(gormutil.OnConflictClause(thing, "primaryKey", true, nil)).
		Create(thing).
		Error
}

func (s *SQL) DeleteStaleThings(ctx context.Context, until time.Time) (int64, error) {
	tx := s.Unmask().WithContext(ctx).
		Where("last_seen < ?", until).
		Delete(new(reddit.ThingData))
	return tx.RowsAffected, tx.Error
}

func (s *SQL) GetPercentile(ctx context.Context, subreddit string, top float64) (int, error) {
	var percentile int
	// language=SQL
	return percentile, s.Unmask().WithContext(ctx).Raw(`
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

func (s *SQL) GetFreshThingIDs(ctx context.Context, subreddit string, ids util.Uint64Set) (util.Uint64Set, error) {
	freshIDs := make([]uint64, 0)
	// language=SQL
	if err := s.Unmask().WithContext(ctx).Raw(`
		select id from reddit
		where subreddit = ? and id in ?
		order by id`,
		subreddit, ids.Slice()).
		Scan(&freshIDs).
		Error; err != nil {
		return nil, err
	}

	set := make(util.Uint64Set, len(freshIDs))
	for _, id := range freshIDs {
		set.Add(id)
	}

	return set, nil
}
