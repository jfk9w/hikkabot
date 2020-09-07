package reddit_test

import (
	"context"
	"testing"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/feed"
	"github.com/jfk9w/hikkabot/vendors/common"
	"github.com/jfk9w/hikkabot/vendors/reddit"
	"github.com/stretchr/testify/assert"
)

type clockMock struct {
	now time.Time
}

func (c *clockMock) Now() time.Time {
	return c.now
}

func TestSQLite3_Basic(t *testing.T) {
	ctx := context.Background()
	clock := &clockMock{now: parseTime(t, "2020-01-01T05:00:00Z")}
	store, err := feed.NewSQLite3(clock, ":memory:")
	assert.Nil(t, err)

	defer store.Close()
	rstore := &reddit.SQLite3{
		SQLite3:  store,
		ThingTTL: 5 * time.Hour,
	}

	assert.Nil(t, rstore.Init(ctx))

	things := []reddit.ThingData{
		{
			Name:      "test1",
			Created:   parseTime(t, "2020-01-01T00:00:00Z"),
			Subreddit: "a",
			Ups:       4,
		},
		{
			Name:      "test2",
			Created:   parseTime(t, "2020-01-01T01:00:00Z"),
			Subreddit: "a",
			Ups:       10,
		},
		{
			Name:      "test3",
			Created:   parseTime(t, "2020-01-01T02:00:00Z"),
			Subreddit: "a",
			Ups:       6,
		},
		{
			Name:      "test4",
			Created:   parseTime(t, "2020-01-01T03:00:00Z"),
			Subreddit: "a",
			Ups:       8,
		},
		{
			Name:      "test5",
			Created:   parseTime(t, "2020-01-01T04:00:00Z"),
			Subreddit: "a",
			Ups:       2,
		},
	}

	data := &reddit.SubredditFeedData{
		Subreddit: "a",
		SentIDs:   make(common.StringSet),
	}

	for _, thing := range things {
		clock.now = thing.Created
		assert.Nil(t, rstore.Thing(ctx, &thing))
		data.SentIDs.Add(thing.Name)
	}

	assertPercentile(t, rstore, "a", 0.8, 4)
	assertPercentile(t, rstore, "a", 0.75, 4)
	assertPercentile(t, rstore, "a", 0.7, 4)
	assertPercentile(t, rstore, "a", 0.6, 6)
	assertPercentile(t, rstore, "a", 0.4, 8)
	assertPercentile(t, rstore, "a", 0.2, 10)
	deleted, err := rstore.Clean(ctx, data)
	assert.Nil(t, err)
	assert.Equal(t, 0, deleted)

	clock.now = clock.now.Add(time.Hour)
	assert.Nil(t, rstore.Thing(ctx, &things[4]))

	assertPercentile(t, rstore, "a", 0.8, 2)
	assertPercentile(t, rstore, "a", 0.75, 6)
	assertPercentile(t, rstore, "a", 0.7, 6)
	assertPercentile(t, rstore, "a", 0.6, 6)
	assertPercentile(t, rstore, "a", 0.4, 8)
	assertPercentile(t, rstore, "a", 0.2, 10)

	deleted, err = rstore.Clean(ctx, data)
	assert.Nil(t, err)
	assert.Equal(t, 1, deleted)

	assert.Nil(t, rstore.Thing(ctx, &things[0]))

	assertPercentile(t, rstore, "a", 0.8, 2)
	assertPercentile(t, rstore, "a", 0.75, 6)
	assertPercentile(t, rstore, "a", 0.7, 6)
	assertPercentile(t, rstore, "a", 0.6, 6)
	assertPercentile(t, rstore, "a", 0.4, 8)
	assertPercentile(t, rstore, "a", 0.2, 10)

	deleted, err = rstore.Clean(ctx, data)
	assert.Nil(t, err)
	assert.Equal(t, 0, deleted)

	clock.now = clock.now.Add(time.Hour)
	assert.Nil(t, rstore.Thing(ctx, &things[4]))

	assertPercentile(t, rstore, "a", 0.8, 2)
	assertPercentile(t, rstore, "a", 0.75, 2)
	assertPercentile(t, rstore, "a", 0.7, 2)
	assertPercentile(t, rstore, "a", 0.6, 6)
	assertPercentile(t, rstore, "a", 0.4, 6)
	assertPercentile(t, rstore, "a", 0.2, 8)

	deleted, err = rstore.Clean(ctx, data)
	assert.Nil(t, err)
	assert.Equal(t, 1, deleted)
}

func assertPercentile(t *testing.T, store reddit.Store, subreddit string, top float64, expected int) {
	percentile, err := store.Percentile(context.Background(), subreddit, top)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, percentile)
}

func parseTime(t *testing.T, str string) time.Time {
	time, err := time.Parse(time.RFC3339, str)
	if err != nil {
		t.Fatal(err)
	}

	return time
}
