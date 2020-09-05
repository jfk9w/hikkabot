package reddit

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jfk9w-go/telegram-bot-api/feed"
	"github.com/pkg/errors"
)

type SQLite3 struct {
	*feed.SQLite3
	ThingTTL      time.Duration
	CleanInterval time.Duration
	lastCleanTime time.Time
}

func (s *SQLite3) Init(ctx context.Context) error {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	  name VARCHAR(63) NOT NULL UNIQUE,
	  subreddit VARCHAR(255) NOT NULL,
	  ups INTEGER NOT NULL,
	  created_at TIMESTAMP NOT NULL
	)`, SQLite3SubredditTable.GetTable())
	if _, err := s.ExecContext(ctx, sql); err != nil {
		return errors.Wrapf(err, "create %s table", SQLite3SubredditTable.GetTable())
	}
	return nil
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
			Where(goqu.C("created_at").Lt(expiry)))
		if err != nil {
			return errors.Wrap(err, "delete")
		}

		s.lastCleanTime = now
		log.Printf("[reddit] deleted %d expired posts", deleted)
	}

	_, err := s.ExecuteSQLBuilder(ctx, s.Database.Insert(SQLite3SubredditTable).
		Cols("subreddit", "name", "created_at", "ups").
		Vals([]interface{}{thing.Subreddit, thing.Name, thing.Created, thing.Ups}).
		OnConflict(goqu.DoUpdate("name", map[string]int{"ups": thing.Ups})))

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
	data.SentNames.ForEach(func(key string) bool {
		if !first {
			values.WriteString(", ")
		} else {
			first = false
		}

		values.WriteString("('")
		values.WriteString(key)
		values.WriteString("')")
		return true
	})
	values.WriteString(")")

	nameColumn := goqu.I("sent_names.name")
	s.RLock()
	rows, err := s.QuerySQLBuilder(ctx, s.Select(nameColumn).
		With("sent_names(name)", goqu.L(values.String())).
		From(goqu.T("sent_names")).
		LeftJoin(SQLite3SubredditTable, goqu.On(nameColumn.Eq(SQLite3SubredditTable.Col("name")))).
		Where(goqu.And(SQLite3SubredditTable.Col("created_at").IsNull())))
	s.RUnlock()
	if err != nil {
		return 0, errors.Wrap(err, "query")
	}

	defer rows.Close()
	removed := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return 0, errors.Wrap(err, "scan")
		}

		data.SentNames.Delete(name)
		removed += 1
	}

	data.LastCleanSecs = now.Unix()
	return removed, nil
}