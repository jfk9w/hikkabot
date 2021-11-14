package feed

import (
	"context"
	"time"

	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/telegram-bot-api"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SQLStorage gorm.DB

func (s *SQLStorage) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQLStorage) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(Subscription))
}

func (s *SQLStorage) Active(ctx context.Context) ([]telegram.ID, error) {
	activeSubs := make([]telegram.ID, 0)
	return activeSubs, s.Unmask().WithContext(ctx).
		Model(new(Subscription)).
		Where("error is null").
		Select("distinct feed_id").
		Scan(&activeSubs).
		Error
}

func (s *SQLStorage) Create(ctx context.Context, sub *Subscription) error {
	tx := s.Unmask().WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Omit("updated_at").
		Create(sub)
	if tx.Error == nil && tx.RowsAffected < 1 {
		return ErrExists
	}

	return tx.Error
}

func (s *SQLStorage) Get(ctx context.Context, header *Header) (*Subscription, error) {
	sub := new(Subscription)
	err := s.Unmask().WithContext(ctx).
		Where("sub_id = ? and vendor = ? and feed_id = ?", header.SubID, header.Vendor, header.FeedID).
		First(sub).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return sub, err
}

func (s *SQLStorage) Shift(ctx context.Context, feedID telegram.ID) (*Subscription, error) {
	sub := new(Subscription)
	err := s.Unmask().WithContext(ctx).
		Where("feed_id = ? and error is null", feedID).
		Order("updated_at asc nulls first").
		First(sub).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}

	return sub, err
}

func (s *SQLStorage) List(ctx context.Context, feedID telegram.ID, active bool) ([]Subscription, error) {
	subs := make([]Subscription, 0)
	return subs, s.Unmask().WithContext(ctx).
		Where("feed_id = ? and (error is null) = ?", feedID, active).
		Find(&subs).
		Error
}

func (s *SQLStorage) DeleteAll(ctx context.Context, feedID telegram.ID, errorLike string) (int64, error) {
	tx := s.Unmask().WithContext(ctx).
		Delete(new(Subscription), "feed_id = ? and error like ?", feedID, errorLike)
	return tx.RowsAffected, tx.Error
}

func (s *SQLStorage) Delete(ctx context.Context, header *Header) error {
	tx := s.Unmask().WithContext(ctx).Delete(&Subscription{Header: header})
	if tx.Error == nil && tx.RowsAffected < 1 {
		return ErrNotFound
	}

	return tx.Error
}

func (s *SQLStorage) Update(ctx context.Context, now time.Time, header *Header, value interface{}) error {
	tx := s.Unmask().WithContext(ctx).
		Model(new(Subscription)).
		Where("sub_id = ? and vendor = ? and feed_id = ?",
			header.SubID, header.Vendor, header.FeedID)

	updates := make(map[string]interface{})
	updates["updated_at"] = now
	switch value := value.(type) {
	case nil:
		tx = tx.Where("error is not null")
		updates["error"] = nil
	case gormf.JSONB:
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
		return ErrNotFound
	}

	return tx.Error
}
