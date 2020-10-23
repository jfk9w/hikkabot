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

type SQLite3 struct {
	*goqu.Database
	flu.Clock
	flu.RWMutex
	metrics.Registry
}

func NewSQLite3(clock flu.Clock, datasource string) (*SQLite3, error) {
	dialect := "sqlite3"
	db, err := sql.Open(dialect, datasource)
	if err != nil {
		return nil, err
	}

	options := sqlite3.DialectOptions()
	options.TimeFormat = "2006-01-02 15:04:05.000"
	goqu.RegisterDialect(dialect, options)

	return &SQLite3{Database: goqu.New(dialect, db), Clock: clock}, nil
}

func (s *SQLite3) Init(ctx context.Context) ([]ID, error) {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	  sub_id TEXT NOT NULL,
	  vendor TEXT NOT NULL,
	  feed_id INTEGER NOT NULL,
      name TEXT NOT NULL,
	  data TEXT,
	  updated_at TIMESTAMP,
	  error VARCHAR(255),
	  UNIQUE(sub_id, vendor, feed_id)
	)`, Table.GetTable())
	if _, err := s.Database.ExecContext(ctx, sql); err != nil {
		return nil, errors.Wrap(err, "create table")
	}
	sql = fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
      feed_id INTEGER NOT NULL,
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

func (s *SQLite3) ExecuteSQLBuilder(ctx context.Context, builder SQLBuilder) (int64, error) {
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

func (s *SQLite3) UpdateSQLBuilder(ctx context.Context, builder SQLBuilder) (bool, error) {
	affected, err := s.ExecuteSQLBuilder(ctx, builder)
	return affected > 0, err
}

func (s *SQLite3) QuerySQLBuilder(ctx context.Context, builder SQLBuilder) (*sql.Rows, error) {
	sql, args, err := builder.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "build sql")
	}

	return s.QueryContext(ctx, sql, args...)
}

func (s *SQLite3) Create(ctx context.Context, sub Sub) error {
	defer s.Lock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, insertSub(s.Insert(Table).OnConflict(goqu.DoNothing()), sub))
	if err == nil && !ok {
		err = ErrExists
	}

	return err
}

func (s *SQLite3) Get(ctx context.Context, id SubID) (Sub, error) {
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

func (s *SQLite3) Advance(ctx context.Context, feedID ID) (Sub, error) {
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

func (s *SQLite3) List(ctx context.Context, feedID ID, active bool) ([]Sub, error) {
	defer s.RLock().Unlock()
	return s.selectSubs(ctx, s.
		From(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.Literal("error IS NULL").Eq(active),
		)))
}

func (s *SQLite3) Clear(ctx context.Context, feedID ID, pattern string) (int64, error) {
	defer s.Lock().Unlock()
	return s.ExecuteSQLBuilder(ctx, s.Database.Delete(Table).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.C("error").Like(pattern),
		)))
}

func (s *SQLite3) Delete(ctx context.Context, id SubID) error {
	defer s.Lock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, s.Database.Delete(Table).Where(s.ByID(id)))
	if err == nil && !ok {
		err = ErrNotFound
	}

	return err
}

var UpdateHistogramBuckets = []float64{1, 10, 100, 500, 1000, 2000, 5000, 10000}

func (s *SQLite3) Update(ctx context.Context, id SubID, value interface{}) error {
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

func (s *SQLite3) Check(ctx context.Context, feedID ID, url string, hash string) error {
	defer s.Lock().Unlock()
	now := s.Now()
	_, err := s.ExecuteSQLBuilder(ctx, s.Insert(BlobTable).
		Cols("feed_id", "url", "hash", "first_seen", "last_seen").
		Vals([]interface{}{feedID, url, hash, now, now}).
		OnConflict(goqu.DoUpdate("feed_id, url",
			map[string]interface{}{
				"collisions": goqu.Literal("collisions + 1"),
				"last_seen":  now,
			})).
		OnConflict(goqu.DoUpdate("feed_id, hash",
			map[string]interface{}{
				"collisions": goqu.Literal("collisions + 1"),
				"last_seen":  now,
			})))
	if err != nil {
		return errors.Wrap(err, "update")
	}

	collisions := 0
	if _, err := s.Select(goqu.MAX(goqu.C("collisions"))).
		From(BlobTable).
		Where(goqu.And(
			goqu.C("feed_id").Eq(feedID),
			goqu.Or(goqu.C("url").Eq(url), goqu.C("hash").Eq(hash)))).
		ScanValContext(ctx, &collisions); err != nil {
		return errors.Wrap(err, "select")
	}

	if collisions > 0 {
		return format.ErrIgnoredMedia
	}

	return nil
}

func (s *SQLite3) Close() error {
	return s.Db.(*sql.DB).Close()
}

func (s *SQLite3) ByID(id SubID) goqu.Expression {
	return goqu.Ex{
		"sub_id":  id.ID,
		"vendor":  id.Vendor,
		"feed_id": id.FeedID,
	}
}

func (s *SQLite3) Lock() flu.Unlocker {
	unlocker := s.RWMutex.Lock()
	return meteredUnlocker{
		Clock:    s.Clock,
		Unlocker: unlocker,
		Registry: s.Registry,
		op:       "write",
		start:    s.Now(),
	}
}

func (s *SQLite3) RLock() flu.Unlocker {
	unlocker := s.RWMutex.RLock()
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
	u.Unlocker.Unlock()
	u.Counter("lock_use_ms",
		metrics.Labels{"op", u.op}).
		Add(float64(u.Now().Sub(u.start).Milliseconds()))
}
