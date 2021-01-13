package feed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/jfk9w/hikkabot/vendors/common"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

var (
	Table     = goqu.T("feed")
	BlobTable = goqu.T("blob")
)

type SQLBuilder interface {
	ToSQL() (string, []interface{}, error)
}

type RWMutex interface {
	Lock() flu.Unlocker
	RLock() flu.Unlocker
}

type SQLStorage struct {
	*goqu.Database
	flu.Clock
	metrics.Registry
}

func NewSQLStorage(clock flu.Clock, driver, conn string) (*SQLStorage, error) {
	db, err := sql.Open(driver, conn)
	if err != nil {
		return nil, err
	}

	if driver != "postgres" {
		panic(errors.New("only postgres supported at the moment"))
	}

	return &SQLStorage{
		Database: goqu.New(driver, db),
		Clock:    clock,
	}, nil
}

func (s *SQLStorage) Init(ctx context.Context) ([]ID, error) {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	  sub_id VARCHAR(255) NOT NULL,
	  vendor VARCHAR(63) NOT NULL,
	  feed_id BIGINT NOT NULL,
      name VARCHAR(255) NOT NULL,
	  data JSONB,
	  updated_at TIMESTAMP,
	  error VARCHAR(255),
	  UNIQUE(sub_id, vendor, feed_id)
	)`, Table.GetTable())
	if _, err := s.Database.ExecContext(ctx, sql); err != nil {
		return nil, errors.Wrap(err, "create table")
	}
	sql = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
      feed_id BIGINT NOT NULL,
	  url VARCHAR(1023) NOT NULL,
      hash_type VARCHAR(15) NOT NULL,
	  hash BYTEA NOT NULL,
	  first_seen TIMESTAMP NOT NULL,
	  last_seen TIMESTAMP,
	  collisions SMALLINT NOT NULL DEFAULT 0,
	  last_url VARCHAR(1023),
	  UNIQUE(feed_id, url),
	  UNIQUE(feed_id, hash_type, hash)
	)`, BlobTable.GetTable())
	if _, err := s.Database.ExecContext(ctx, sql); err != nil {
		return nil, errors.Wrap(err, "create blob table")
	}
	activeSubs := make([]ID, 0)
	err := s.Select(goqu.DISTINCT("feed_id")).
		From(Table).
		Where(goqu.C("error").IsNull()).
		ScanValsContext(ctx, &activeSubs)
	if err != nil {
		return nil, errors.Wrap(err, "select active subs")
	}
	return activeSubs, nil
}

func (s *SQLStorage) ExecuteSQLBuilder(ctx context.Context, builder SQLBuilder) (int64, error) {
	sql, args, err := builder.ToSQL()
	if err != nil {
		return 0, errors.Wrap(err, "build sql")
	}
	result, err := s.Database.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "execute")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "rows affected")
	}
	return affected, nil
}

func (s *SQLStorage) UpdateSQLBuilder(ctx context.Context, builder SQLBuilder) (bool, error) {
	affected, err := s.ExecuteSQLBuilder(ctx, builder)
	return affected > 0, err
}

func (s *SQLStorage) QuerySQLBuilder(ctx context.Context, builder SQLBuilder) (*sql.Rows, error) {
	sql, args, err := builder.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "build sql")
	}

	return s.QueryContext(ctx, sql, args...)
}

var subColumnOrder = []interface{}{
	"sub_id",
	"vendor",
	"feed_id",
	"name",
	"data",
	"updated_at",
}

func (s *SQLStorage) selectSubs(ctx context.Context, builder *goqu.SelectDataset) ([]Sub, error) {
	rows, err := s.QuerySQLBuilder(ctx, builder.Select(subColumnOrder...))
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}

	defer rows.Close()
	subs := make([]Sub, 0)
	for rows.Next() {
		sub := Sub{}
		if err := rows.Scan(
			&sub.SubID.ID, &sub.SubID.Vendor, &sub.SubID.FeedID,
			&sub.Name, &sub.Data, &sub.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "scan")
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func (s *SQLStorage) CreateSub(ctx context.Context, sub Sub) error {
	defer s.Lock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, s.Insert(Table).OnConflict(goqu.DoNothing()).
		Cols(subColumnOrder...).
		Vals([]interface{}{
			sub.SubID.ID, sub.SubID.Vendor, sub.SubID.FeedID,
			sub.Name, sub.Data, sub.UpdatedAt,
		}))

	if err == nil && !ok {
		err = ErrExists
	}

	return err
}

func (s *SQLStorage) GetSub(ctx context.Context, id SubID) (Sub, error) {
	defer s.RLock().Unlock()
	subs, err := s.selectSubs(ctx, s.
		From(Table).
		Where(s.ByID(id)).
		Limit(1))
	if err != nil {
		return Sub{}, errors.Wrap(err, "select")
	}

	if len(subs) == 0 {
		return Sub{}, ErrNotFound
	}

	return subs[0], nil
}

func (s *SQLStorage) NextSub(ctx context.Context, feedID ID) (Sub, error) {
	defer s.RLock().Unlock()
	subs, err := s.selectSubs(ctx, s.
		From(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.C("error").IsNull(),
		)).
		Order(goqu.I("updated_at").Asc().NullsFirst()).
		Limit(1))
	if err != nil {
		return Sub{}, errors.Wrap(err, "select")
	}

	if len(subs) == 0 {
		return Sub{}, ErrNotFound
	}

	return subs[0], nil
}

func (s *SQLStorage) ListSubs(ctx context.Context, feedID ID, active bool) ([]Sub, error) {
	defer s.RLock().Unlock()
	return s.selectSubs(ctx, s.
		From(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.Literal("error IS NULL").Eq(active),
		)))
}

func (s *SQLStorage) DeleteSubs(ctx context.Context, feedID ID, errorLike string) (int64, error) {
	defer s.Lock().Unlock()
	return s.ExecuteSQLBuilder(ctx, s.Database.Delete(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.C("error").Like(errorLike),
		)))
}

func (s *SQLStorage) DeleteSub(ctx context.Context, id SubID) error {
	defer s.Lock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, s.Database.Delete(Table).Where(s.ByID(id)))
	if err == nil && !ok {
		err = ErrNotFound
	}

	return err
}

func (s *SQLStorage) UpdateSub(ctx context.Context, id SubID, value interface{}) error {
	defer s.Lock().Unlock()

	where := s.ByID(id)
	update := goqu.Record{"updated_at": s.Now()}
	switch value := value.(type) {
	case nil:
		where = goqu.And(where, goqu.C("error").IsNotNull())
		update["error"] = nil
	case Data:
		where = goqu.And(where, goqu.C("error").IsNull())
		update["data"] = value
	case error:
		where = goqu.And(where, goqu.C("error").IsNull())
		update["error"] = value.Error()
	default:
		return errors.Errorf("invalid update value type: %T", value)
	}

	ok, err := s.UpdateSQLBuilder(ctx, s.Database.Update(Table).Set(update).Where(where))
	if err == nil && !ok {
		err = ErrNotFound
	}

	return err
}

func (s *SQLStorage) CheckBlob(ctx context.Context, feedID ID, url string, hashType string, hash []byte) error {
	defer s.Lock().Unlock()
	now := s.Now().In(time.UTC)

	hashValue := fmt.Sprintf(`%x`, hash)
	if updated, err := s.ExecuteSQLBuilder(ctx, s.Insert(BlobTable).
		Cols("feed_id", "url", "hash_type", "hash", "first_seen").
		Vals([]interface{}{feedID, url, hashType, hashValue, now}).
		OnConflict(goqu.DoNothing())); err != nil {
		return errors.Wrap(err, "update")
	} else if updated > 0 {
		return nil
	}

	update := common.PlainSQLBuilder{
		SQL: fmt.Sprintf(
			"UPDATE %s SET collisions = collisions + 1, last_seen = $1, last_url = $2 WHERE url = $2 OR hash = $3 RETURNING url",
			BlobTable.GetTable()),
		Arguments: []interface{}{now, url, hashValue},
	}

	rows, err := s.QuerySQLBuilder(ctx, update)
	if err != nil {
		return errors.Wrap(err, "update collision")
	}

	defer rows.Close()
	oldURL := ""
	for rows.Next() {
		if err := rows.Scan(&oldURL); err != nil {
			return errors.Wrap(err, "scan")
		}
	}

	if oldURL != url {
		return errors.Wrapf(format.ErrSkipMedia, "duplicates %s", oldURL)
	} else {
		return errors.Wrap(format.ErrSkipMedia, "duplicate")
	}
}

func (s *SQLStorage) BlobPage(ctx context.Context, feedID ID, hashType string, offset, limit uint) ([]string, error) {
	urls := make([]string, 0)
	if err := s.Database.Select(goqu.C("url")).
		From(BlobTable).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.C("hash_type").Eq(hashType))).
		Order(goqu.C("collisions").Desc(), goqu.C("url").Asc()).
		Offset(offset).
		Limit(limit).
		ScanValsContext(ctx, &urls); err != nil {
		return nil, errors.Wrap(err, "select blobs")
	}

	return urls, nil
}

func (s *SQLStorage) Close() error {
	return s.Db.(*sql.DB).Close()
}

func (s *SQLStorage) ByID(id SubID) goqu.Expression {
	return goqu.Ex{
		"sub_id":  id.ID,
		"vendor":  id.Vendor,
		"feed_id": id.FeedID,
	}
}

func (s *SQLStorage) Lock() flu.Unlocker {
	return meteredUnlocker{
		Clock:    s.Clock,
		Registry: s.Registry,
		op:       "write",
		start:    s.Now(),
	}
}

func (s *SQLStorage) RLock() flu.Unlocker {
	return meteredUnlocker{
		Clock:    s.Clock,
		Registry: s.Registry,
		op:       "read",
		start:    s.Now(),
	}
}

type meteredUnlocker struct {
	flu.Clock
	metrics.Registry
	op    string
	start time.Time
}

func (u meteredUnlocker) Unlock() {
	u.Counter("lock_use_ms",
		metrics.Labels{"op", u.op}).
		Add(float64(u.Now().Sub(u.start).Milliseconds()))
}
