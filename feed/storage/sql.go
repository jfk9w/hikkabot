package storage

import (
	"context"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"
	"github.com/jfk9w/hikkabot/feed"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SQL gorm.DB

func (s *SQL) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQL) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).
		AutoMigrate(new(feed.Subscription), new(feed.BlobHash))
}

func (s *SQL) Active(ctx context.Context) ([]telegram.ID, error) {
	activeSubs := make([]telegram.ID, 0)
	return activeSubs, s.Unmask().WithContext(ctx).
		Model(new(feed.Subscription)).
		Where("error is null").
		Select("distinct feed_id").
		Scan(&activeSubs).
		Error
}

func (s *SQL) Create(ctx context.Context, sub *feed.Subscription) error {
	tx := s.Unmask().WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(sub)
	if tx.Error == nil && tx.RowsAffected < 1 {
		return feed.ErrExists
	}

	return tx.Error
}

func (s *SQL) Get(ctx context.Context, header *feed.Header) (*feed.Subscription, error) {
	sub := new(feed.Subscription)
	err := s.Unmask().WithContext(ctx).
		Where("sub_id = ? and vendor = ? and feed_id = ?", header.SubID, header.Vendor, header.FeedID).
		First(sub).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, feed.ErrNotFound
	}

	return sub, err
}

func (s *SQL) Shift(ctx context.Context, feedID telegram.ID) (*feed.Subscription, error) {
	sub := new(feed.Subscription)
	err := s.Unmask().WithContext(ctx).
		Where("feed_id = ? and error is null", feedID).
		Order("updated_at asc nulls first").
		First(sub).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, feed.ErrNotFound
	}

	return sub, err
}

func (s *SQL) List(ctx context.Context, feedID telegram.ID, active bool) ([]feed.Subscription, error) {
	subs := make([]feed.Subscription, 0)
	return subs, s.Unmask().WithContext(ctx).
		Where("feed_id = ? and (error is null) = ?", feedID, active).
		Find(&subs).
		Error
}

func (s *SQL) DeleteAll(ctx context.Context, feedID telegram.ID, errorLike string) (int64, error) {
	tx := s.Unmask().WithContext(ctx).
		Where("feed_id = ? and error like ?", feedID, errorLike).
		Delete(new(feed.Subscription))
	return tx.RowsAffected, tx.Error
}

func (s *SQL) Delete(ctx context.Context, header *feed.Header) error {
	tx := s.Unmask().WithContext(ctx).Delete(&feed.Subscription{Header: header})
	if tx.Error == nil && tx.RowsAffected < 1 {
		return feed.ErrNotFound
	}

	return tx.Error
}

func (s *SQL) Update(ctx context.Context, now time.Time, header *feed.Header, value interface{}) error {
	tx := s.Unmask().WithContext(ctx).
		Model(new(feed.Subscription)).
		Where("sub_id = ? and vendor = ? and feed_id = ?",
			header.SubID, header.Vendor, header.FeedID)

	updates := make(map[string]interface{})
	updates["updated_at"] = now
	switch value := value.(type) {
	case nil:
		tx = tx.Where("error is not null")
		updates["error"] = nil
	case gormutil.JSONB:
		tx = tx.Where("error is null")
		updates["data"] = value
	case error:
		tx = tx.Where("error is null")
		updates["error"] = value.Error()
	default:
		return errors.Errorf("invalid update value type: %T", value)
	}

	tx = tx.UpdateColumns(updates)
	if tx.Error == nil && tx.RowsAffected < 1 {
		return feed.ErrNotFound
	}

	return tx.Error
}

func (s *SQL) Check(ctx context.Context, hash *feed.BlobHash) error {
	update := clause.Set{
		clause.Assignment{Column: clause.Column{Name: "collisions"}, Value: gorm.Expr("blob.collisions + 1")},
		clause.Assignment{Column: clause.Column{Name: "url"}, Value: hash.URL},
		clause.Assignment{Column: clause.Column{Name: "hash_type"}, Value: hash.Type},
		clause.Assignment{Column: clause.Column{Name: "hash"}, Value: hash.Hash},
		clause.Assignment{Column: clause.Column{Name: "last_seen"}, Value: hash.LastSeen},
	}

	err := s.Unmask().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(gormutil.OnConflictClause(hash, "primaryKey", false, update)).
			Create(hash).
			Error; err != nil {
			return errors.Wrap(err, "create")
		}

		if err := tx.First(hash).Error; err != nil {
			return errors.Wrap(err, "find")
		}

		return nil
	})

	if err == nil && hash.Collisions > 0 {
		err = richtext.ErrSkipMedia
	}

	return err
}
