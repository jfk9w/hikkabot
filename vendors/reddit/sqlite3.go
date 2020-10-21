package reddit

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/pkg/errors"
)

type SQLite3 struct {
	*feed.SQLite3
	ThingTTL      time.Duration
	CleanInterval time.Duration
	lastCleanTime time.Time
}

func (s *SQLite3) Init(ctx context.Context) (Store, error) {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	  id VARCHAR(63) NOT NULL UNIQUE,
	  subreddit VARCHAR(255) NOT NULL,
	  ups INTEGER NOT NULL,
	  last_seen TIMESTAMP NOT NULL
	)`, SQLite3SubredditTable.GetTable())
	if _, err := s.ExecContext(ctx, sql); err != nil {
		return nil, errors.Wrapf(err, "create %s table", SQLite3SubredditTable.GetTable())
	}
	return s, nil
}

func (s *SQLite3) Thing(ctx context.Context, thing *ThingData) error {
	if s.Now().Sub(thing.Created) > s.ThingTTL {
		return nil
	}

	defer s.Lock().Unlock()
	if s.Now().Sub(s.lastCleanTime) > s.CleanInterval {
		now := s.Now()
		expiry := now.Add(-s.ThingTTL)
		deleted, err := s.ExecuteSQLBuilder(ctx, s.Database.Delete(SQLite3SubredditTable).
			Where(goqu.C("last_seen").Lt(expiry)))
		if err != nil {
			return errors.Wrap(err, "delete")
		}

		s.lastCleanTime = now
		log.Printf("[reddit] deleted %d expired posts", deleted)
	}

	_, err := s.ExecuteSQLBuilder(ctx, s.Database.Insert(SQLite3SubredditTable).
		Cols("subreddit", "id", "last_seen", "ups").
		Vals([]interface{}{thing.Subreddit, strconv.FormatUint(thing.ID, 36), s.Now(), thing.Ups}).
		OnConflict(goqu.DoUpdate("id", map[string]int{"ups": thing.Ups})))

	return err
}

func (s *SQLite3) Percentile(ctx context.Context, subreddit string, top float64) (int, error) {
	defer s.RLock().Unlock()
	var percentile int
	ok, err := s.
		Select(goqu.C("ups")).
		From(goqu.Select(
			goqu.C("ups"),
			goqu.CUME_DIST().Over(goqu.W().OrderBy(goqu.C("ups").Desc())).As("rank")).
			From(SQLite3SubredditTable).
			Where(goqu.C("subreddit").Eq(subreddit)).
			As("ranking")).
		Where(goqu.C("rank").Gte(top)).
		Order(goqu.C("rank").Asc()).
		ScanValContext(ctx, &percentile)

	if err != nil {
		return 0, errors.Wrap(err, "scan from db")
	}

	if !ok {
		return -1, nil
	}

	return percentile, nil
}

func (s *SQLite3) Clean(ctx context.Context, data *SubredditFeedData) (int, error) {
	lastClean := time.Unix(data.LastCleanSecs, 0)
	now := s.Now()
	if now.Sub(lastClean) < s.CleanInterval {
		return 0, nil
	}

	var values strings.Builder
	values.WriteString("(VALUES ")
	first := true
	for value := range data.SentIDs {
		if !first {
			values.WriteString(", ")
		} else {
			first = false
		}

		values.WriteString("('")
		values.WriteString(strconv.FormatUint(value, 36))
		values.WriteString("')")
	}
	values.WriteString(")")

	nameColumn := goqu.I("sent_ids.id")
	unlocker := s.RLock()
	rows, err := s.QuerySQLBuilder(ctx, s.Select(nameColumn).
		With("sent_ids(id)", goqu.L(values.String())).
		From(goqu.T("sent_ids")).
		LeftJoin(SQLite3SubredditTable, goqu.On(nameColumn.Eq(SQLite3SubredditTable.Col("id")))).
		Where(goqu.And(SQLite3SubredditTable.Col("last_seen").IsNull())))
	unlocker.Unlock()
	if err != nil {
		return 0, errors.Wrap(err, "query")
	}

	defer rows.Close()
	removed := 0
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return 0, errors.Wrap(err, "scan")
		}

		delete(data.SentIDs, id)
		removed += 1
	}

	data.LastCleanSecs = now.Unix()
	return removed, nil
}
