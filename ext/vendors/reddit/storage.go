package reddit

import (
	"context"
	_ "embed"
	"time"

	"hikkabot/3rdparty/reddit"
	"hikkabot/core"
	"hikkabot/feed"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/gormf"
	"gorm.io/gorm"
)

//go:embed subreddit_index.sql
var subredditIndexSQL string

const storageServiceID = "vendors.reddit.storage"

type Storage[C core.StorageContext] struct {
	StorageInterface
}

func (s Storage[C]) String() string {
	return storageServiceID
}

func (s *Storage[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	if s.StorageInterface != nil {
		return nil
	}

	var storage core.Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	db := &apfel.GormDB[C]{Config: app.Config().StorageConfig()}
	if err := app.Use(ctx, db, false); err != nil {
		return err
	}

	if err := db.DB().WithContext(ctx).AutoMigrate(new(reddit.Thing)); err != nil {
		return err
	}

	if err := db.DB().WithContext(ctx).Raw(subredditIndexSQL).Error; err != nil {
		return errors.Wrap(err, "create indices")
	}

	s.StorageInterface = &sqlStorage{
		Storage:      storage,
		EventStorage: storage,
		db:           db.DB(),
	}

	return nil
}

func (s *sqlStorage) SaveThings(ctx context.Context, things []reddit.Thing) error {
	return s.db.WithContext(ctx).
		Clauses(gormf.OnConflictClause(things, "primaryKey", true, nil)).
		Create(things).
		Error
}

type sqlStorage struct {
	feed.Storage
	feed.EventStorage
	db *gorm.DB
}

func (s *sqlStorage) RedditTx(ctx context.Context, body func(tx StorageTx) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return body(&sqlStorageTx{db: s.db}) })
}

type sqlStorageTx struct {
	db *gorm.DB
}

func (stx *sqlStorageTx) GetPercentile(subreddit string, top float64) (int, error) {
	var percentile int
	return percentile, stx.db.Raw( /* language=SQL */ `
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

func (stx *sqlStorageTx) Score(feedID feed.ID, thingIDs []string) (*Score, error) {
	score := new(Score)
	return score, stx.db.Raw( /* language=SQL */ `
		select min(time) as first,
		       count(distinct case when type in ('click', 'like') then jsonb_extract_path_text(data, 'thing_id') end) as liked_things,
		       count(distinct case when type = 'dislike' then jsonb_extract_path(data, 'user_id') end) as disliked_things,
		       count(distinct case when type in ('click', 'like') then jsonb_extract_path(data, 'user_id') end) as likes,
		       count(distinct case when type = 'dislike' then jsonb_extract_path(data, 'user_id') end) as dislikes
		from event
		where chat_id = ? and jsonb_extract_path_text(data, 'thing_id') in ?`,
		feedID, thingIDs).
		Scan(score).
		Error
}

func (stx *sqlStorageTx) DeleteStaleThings(until time.Time) (int64, error) {
	tx := stx.db.
		Where("last_seen < ?", until).
		Delete(new(reddit.Thing))
	return tx.RowsAffected, tx.Error
}

func (stx *sqlStorageTx) GetFreshThingIDs(ids flu.Set[string]) (flu.Set[string], error) {
	freshIDs := make(flu.Slice[string], 0)
	if err := stx.db.Raw( /* language=SQL */ `
		select id from reddit
		where id in ?
		order by num_id`,
		flu.ToSlice[string](ids)).
		Scan(&freshIDs).
		Error; err != nil {
		return nil, err
	}

	set := make(flu.Set[string], len(freshIDs))
	flu.AppendAll[string](&set, freshIDs)
	return set, nil
}
