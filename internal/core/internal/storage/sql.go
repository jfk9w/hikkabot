package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jfk9w-go/flu/colf"
	"github.com/jfk9w-go/flu/gormf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w/hikkabot/internal/feed"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SQL struct {
	Clock syncf.Clock
	DB    *gorm.DB
	IsPG  bool
}

func (s *SQL) GetActiveFeedIDs(ctx context.Context) ([]feed.ID, error) {
	feedIDs := make([]feed.ID, 0)
	return feedIDs, s.DB.WithContext(ctx).
		Model(new(feed.Subscription)).
		Where("error is null").
		Select("distinct feed_id").
		Scan(&feedIDs).
		Error
}

func (s *SQL) GetSubscription(ctx context.Context, header feed.Header) (*feed.Subscription, error) {
	return (&sqlTx{db: s.DB.WithContext(ctx)}).GetSubscription(header)
}

func (s *SQL) CreateSubscription(ctx context.Context, sub *feed.Subscription) error {
	tx := s.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Omit("updated_at").
		Create(sub)
	if tx.Error == nil && tx.RowsAffected < 1 {
		return errors.New("exists")
	}

	return tx.Error
}

func (s *SQL) ShiftSubscription(ctx context.Context, feedID feed.ID) (*feed.Subscription, error) {
	var sub feed.Subscription
	err := s.DB.WithContext(ctx).
		Where("feed_id = ? and error is null", feedID).
		Order("updated_at asc nulls first").
		First(&sub).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, feed.ErrNotFound
	}

	return &sub, err
}

func (s *SQL) ListSubscriptions(ctx context.Context, feedID feed.ID, active bool) ([]feed.Subscription, error) {
	var subs []feed.Subscription
	return subs, s.DB.WithContext(ctx).
		Where("feed_id = ? and (error is null) = ? and (error is null or error != ?)", feedID, active, feed.Deadborn).
		Find(&subs).
		Error
}

func (s *SQL) DeleteAllSubscriptions(ctx context.Context, feedID feed.ID, errorLike string) (int64, error) {
	tx := s.DB.WithContext(ctx).
		Delete(new(feed.Subscription), "feed_id = ? and error like ?", feedID, errorLike)
	return tx.RowsAffected, tx.Error
}

func (s *SQL) UpdateSubscription(ctx context.Context, header feed.Header, value any) error {
	tx := &sqlTx{
		clock: s.Clock,
		db:    s.DB.WithContext(ctx),
	}

	return tx.UpdateSubscription(header, value)
}

func (s *SQL) Tx(ctx context.Context, body func(tx feed.Tx) error) error {
	return s.tx(ctx, func(tx *gorm.DB) error { return body(&sqlTx{clock: s.Clock, db: s.DB}) })
}

func (s *SQL) SaveEvent(ctx context.Context, feedID feed.ID, eventType string, value any) error {
	return (&sqlTx{clock: s.Clock, db: s.DB.WithContext(ctx)}).SaveEvent(feedID, eventType, value)
}

func (s *SQL) CountEventsBy(ctx context.Context, feedID feed.ID, since time.Time, key string, multipliers map[string]float64) (map[string]int64, error) {
	if err := postgresDisclaimer(s.IsPG, "CountEventsBy"); err != nil {
		return nil, err
	}

	var rows []struct {
		Type   string
		Key    string
		Events int64
	}

	types := colf.Keys[string, float64](multipliers)
	if err := s.DB.WithContext(ctx).Raw(fmt.Sprintf( /* language=SQL */ `
		select type, jsonb_extract_path_text(data, '%s') as key, count(1) as events
		from event
		where chat_id = ? and type in ? and time >= ? 
		group by 1, 2`, key),
		feedID, types, since).
		Scan(&rows).
		Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, row := range rows {
		stats[row.Key] += int64(float64(row.Events) * multipliers[row.Type])
	}

	return stats, nil
}

func (s *SQL) EventTx(ctx context.Context, body func(tx feed.EventTx) error) error {
	return s.tx(ctx, func(tx *gorm.DB) error { return body(&sqlTx{clock: s.Clock, db: s.DB, isPG: s.IsPG}) })
}

func (s *SQL) IsMediaUnique(ctx context.Context, hash *feed.MediaHash) (bool, error) {
	update := clause.Set{
		clause.Assignment{Column: clause.Column{Name: "collisions"}, Value: gorm.Expr("blob.collisions + 1")},
		clause.Assignment{Column: clause.Column{Name: "url"}, Value: hash.URL},
		clause.Assignment{Column: clause.Column{Name: "hash_type"}, Value: hash.Type},
		clause.Assignment{Column: clause.Column{Name: "hash"}, Value: hash.Value},
		clause.Assignment{Column: clause.Column{Name: "last_seen"}, Value: hash.LastSeen},
	}

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(gormf.OnConflictClause(hash, "primaryKey", false, update)).
			Create(hash).
			Error; err != nil {
			return errors.Wrap(err, "create")
		}

		if err := tx.First(hash).Error; err != nil {
			return errors.Wrap(err, "find")
		}

		return nil
	})

	ok := false
	if err == nil && hash.Collisions == 0 {
		ok = true
	}

	return ok, err
}

func (s *SQL) tx(ctx context.Context, body func(tx *gorm.DB) error) error {
	return s.DB.WithContext(ctx).Transaction(body)
}

type sqlTx struct {
	clock syncf.Clock
	db    *gorm.DB
	isPG  bool
}

func (stx *sqlTx) GetSubscription(header feed.Header) (*feed.Subscription, error) {
	var sub feed.Subscription
	err := stx.db.
		Where("lower(sub_id) = lower(?) and vendor = ? and feed_id = ?", header.SubID, header.Vendor, header.FeedID).
		First(&sub).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, feed.ErrNotFound
	}

	return &sub, err
}

func (stx *sqlTx) DeleteSubscription(header feed.Header) error {
	tx := stx.db.Delete(&feed.Subscription{Header: header})
	if tx.Error == nil && tx.RowsAffected < 1 {
		return feed.ErrNotFound
	}

	return tx.Error
}

func (stx *sqlTx) UpdateSubscription(header feed.Header, value interface{}) error {
	tx := stx.db.
		Model(new(feed.Subscription)).
		Where("sub_id = ? and vendor = ? and feed_id = ?",
			header.SubID, header.Vendor, header.FeedID)

	updates := make(map[string]interface{})
	updates["updated_at"] = stx.clock.Now()
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
		return feed.ErrNotFound
	}

	return tx.Error
}

func (stx *sqlTx) GetLastEventData(feedID feed.ID, eventType string, filter map[string]any, value any) error {
	if err := postgresDisclaimer(stx.isPG, "GetLastEventData"); err != nil {
		return err
	}

	where, values := whereEvent(feedID, []string{eventType}, filter)
	var row struct {
		Data gormf.JSONB
	}

	if err := stx.db.Model(new(feed.Event)).
		Where(where, values...).
		Order("time desc").
		Limit(1).
		Select("data").
		Scan(&row).
		Error; err != nil {
		return err
	}

	return row.Data.As(value)
}

func (stx *sqlTx) SaveEvent(feedID feed.ID, eventType string, value any) error {
	data, err := gormf.ToJSONB(value)
	if err != nil {
		return err
	}

	event := &feed.Event{
		Time:   stx.clock.Now(),
		Type:   eventType,
		FeedID: feedID,
		Data:   data,
	}

	return stx.db.Create(event).Error
}

func (stx *sqlTx) DeleteEvents(feedID feed.ID, types []string, filter map[string]any) error {
	if err := postgresDisclaimer(stx.isPG, "DeleteEvents"); err != nil {
		return err
	}

	where, values := whereEvent(feedID, types, filter)
	return stx.db.
		Delete(new(feed.Event), append([]any{where}, values...)...).
		Error
}

func (stx *sqlTx) CountEventsByType(feedID feed.ID, types []string, filter map[string]any) (map[string]int64, error) {
	if err := postgresDisclaimer(stx.isPG, "CountEventsByType"); err != nil {
		return nil, err
	}

	var rows []struct {
		Type   string
		Events int64
	}

	where, values := whereEvent(feedID, types, filter)
	if err := stx.db.Raw(fmt.Sprintf( /* language=SQL */ `
		select type, count(1) as events
		from event
		where %s
		group by type`, where),
		values...).
		Scan(&rows).
		Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, row := range rows {
		stats[row.Type] = row.Events
	}

	return stats, nil
}

func postgresDisclaimer(isPG bool, name string) error {
	if !isPG {
		logf.Get(ServiceID).Warnf(context.TODO(), "%s is not supported, you may want to switch to postgres", name)
		return feed.ErrUnsupported
	}

	return nil
}

func whereEvent(feedID feed.ID, types []string, filter map[string]any) (string, []any) {
	var where strings.Builder
	where.WriteString("chat_id = ? and type in ?")
	values := []any{feedID, types}
	for key, value := range filter {
		where.WriteString(fmt.Sprintf(` and jsonb_extract_path_text(data, '%s') = ?::text`, key))
		values = append(values, value)
	}

	return where.String(), values
}
