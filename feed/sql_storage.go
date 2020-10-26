package feed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/dialect/sqlite3"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/metrics"
	"github.com/jfk9w-go/telegram-bot-api/format"
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
	RWMutex
	metrics.Registry
	driver string
}

func NewSQLStorage(clock flu.Clock, driver, conn string) (*SQLStorage, error) {
	db, err := sql.Open(driver, conn)
	if err != nil {
		return nil, err
	}

	var mutex RWMutex
	if driver == "sqlite3" {
		options := sqlite3.DialectOptions()
		options.TimeFormat = "2006-01-02 15:04:05.000"
		goqu.RegisterDialect("sqlite3", options)
		mutex = new(flu.RWMutex)
	}

	return &SQLStorage{
		Database: goqu.New(driver, db),
		Clock:    clock,
		RWMutex:  mutex,
		driver:   driver,
	}, nil
}

func (s *SQLStorage) Init(ctx context.Context) ([]ID, error) {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	  sub_id TEXT NOT NULL,
	  vendor TEXT NOT NULL,
	  feed_id BIGINT NOT NULL,
      name TEXT NOT NULL,
	  data JSON,
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
	  url TEXT NOT NULL,
	  hash TEXT NOT NULL,
	  first_seen TIMESTAMP NOT NULL,
	  last_seen TIMESTAMP,
	  collisions INTEGER NOT NULL DEFAULT 0,
	  UNIQUE(feed_id, url),
	  UNIQUE(feed_id, hash)
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

func (s *SQLStorage) Create(ctx context.Context, sub Sub) error {
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

func (s *SQLStorage) Get(ctx context.Context, id SubID) (Sub, error) {
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

func (s *SQLStorage) Advance(ctx context.Context, feedID ID) (Sub, error) {
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

func (s *SQLStorage) List(ctx context.Context, feedID ID, active bool) ([]Sub, error) {
	defer s.RLock().Unlock()
	return s.selectSubs(ctx, s.
		From(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.Literal("error IS NULL").Eq(active),
		)))
}

func (s *SQLStorage) Clear(ctx context.Context, feedID ID, pattern string) (int64, error) {
	defer s.Lock().Unlock()
	return s.ExecuteSQLBuilder(ctx, s.Database.Delete(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.C("error").Like(pattern),
		)))
}

func (s *SQLStorage) Delete(ctx context.Context, id SubID) error {
	defer s.Lock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, s.Database.Delete(Table).Where(s.ByID(id)))
	if err == nil && !ok {
		err = ErrNotFound
	}

	return err
}

func (s *SQLStorage) Update(ctx context.Context, id SubID, value interface{}) error {
	defer s.Lock().Unlock()

	where := s.ByID(id)
	update := map[string]interface{}{"updated_at": s.Now()}
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

func (s *SQLStorage) Check(ctx context.Context, feedID ID, url string, hash string) error {
	defer s.Lock().Unlock()
	now := s.Now()

	columnUpdatePrefix := ""
	if s.driver == "postgres" {
		columnUpdatePrefix = "excluded."
	}

	_, err := s.ExecuteSQLBuilder(ctx, s.Insert(BlobTable).
		Cols("feed_id", "url", "hash", "first_seen", "last_seen").
		Vals([]interface{}{feedID, url, hash, now, now}).
		OnConflict(goqu.DoUpdate("feed_id, url",
			map[string]interface{}{
				"collisions": goqu.Literal(columnUpdatePrefix + `collisions + 1`),
				"last_seen":  now,
			})).
		OnConflict(goqu.DoUpdate("feed_id, hash",
			map[string]interface{}{
				"collisions": goqu.Literal(columnUpdatePrefix + `collisions + 1`),
				"last_seen":  now,
			})))
	if err != nil {
		return errors.Wrap(err, "update")
	}

	rows, err := s.QuerySQLBuilder(ctx, s.Select(goqu.C("url"), goqu.C("collisions")).
		From(BlobTable).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.Or(
				goqu.C("url").Eq(url),
				goqu.C("hash").Eq(hash)))).
		Limit(1))
	if err != nil {
		return errors.Wrap(err, "select")
	}

	defer rows.Close()
	collisions := 0
	oldURL := ""
	for rows.Next() {
		if err := rows.Scan(&oldURL, &collisions); err != nil {
			return errors.Wrap(err, "scan")
		}
	}

	if collisions > 0 {
		return errors.Wrapf(format.ErrIgnoredMedia, "duplicates %s", oldURL)
	}

	return nil
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
	var unlocker flu.Unlocker
	if s.RWMutex != nil {
		unlocker = s.RWMutex.Lock()
	}

	return meteredUnlocker{
		Clock:    s.Clock,
		Unlocker: unlocker,
		Registry: s.Registry,
		op:       "write",
		start:    s.Now(),
	}
}

func (s *SQLStorage) RLock() flu.Unlocker {
	var unlocker flu.Unlocker
	if s.RWMutex != nil {
		unlocker = s.RWMutex.RLock()
	}

	return meteredUnlocker{
		Clock:    s.Clock,
		Unlocker: unlocker,
		Registry: s.Registry,
		op:       "read",
		start:    s.Now(),
	}
}

type meteredUnlocker struct {
	flu.Clock
	flu.Unlocker
	metrics.Registry
	op    string
	start time.Time
}

func (u meteredUnlocker) Unlock() {
	if u.Unlocker != nil {
		u.Unlocker.Unlock()
	}

	u.Counter("lock_use_ms",
		metrics.Labels{"op", u.op}).
		Add(float64(u.Now().Sub(u.start).Milliseconds()))
}
