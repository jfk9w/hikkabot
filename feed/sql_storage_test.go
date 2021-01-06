package feed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jfk9w-go/flu"

	"github.com/jfk9w/hikkabot/feed"
	"github.com/stretchr/testify/assert"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time {
	return c.now
}

func newTestSQLite3(t *testing.T, clock flu.Clock) *feed.SQLStorage {
	store, err := feed.NewSQLStorage(clock, "sqlite3", ":memory:")
	assert.Nil(t, err)
	return store
}

func TestSQLite3_Basic(t *testing.T) {
	clock := new(testClock)
	store := newTestSQLite3(t, clock)
	defer store.Close()

	ctx := context.Background()
	activeSubs, err := store.Init(ctx)
	assert.Nil(t, err)
	assert.Empty(t, activeSubs)

	sub := feed.Sub{
		SubID: feed.SubID{"1", "test", 1},
		Name:  "test feed",
		Data:  feed.Data(`{"value": 5}`),
	}

	_, err = store.GetSub(ctx, sub.SubID)
	assert.Equal(t, feed.ErrNotFound, err)
	err = store.CreateSub(ctx, sub)
	assert.Nil(t, err)
	err = store.CreateSub(ctx, sub)
	assert.Equal(t, feed.ErrExists, err)
	stored, err := store.GetSub(ctx, sub.SubID)
	assert.Nil(t, err)
	assert.Equal(t, sub, stored)
	stored, err = store.NextSub(ctx, sub.FeedID)
	assert.Nil(t, err)
	assert.Equal(t, sub, stored)
	list, err := store.ListSubs(ctx, sub.FeedID, true)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Sub{sub}, list)
	clock.now = time.Date(2020, 8, 13, 13, 54, 64, 0, time.UTC)
	sub.UpdatedAt = &clock.now
	data, err := feed.DataFrom(struct{ field string }{"value"})
	assert.Nil(t, err)
	sub.Data = data
	err = store.UpdateSub(ctx, sub.SubID, data)
	assert.Nil(t, err)
	stored, err = store.GetSub(ctx, sub.SubID)
	assert.Nil(t, err)
	assert.Equal(t, sub, stored)
	err = store.UpdateSub(ctx, sub.SubID, errors.New("test error"))
	assert.Nil(t, err)
	list, err = store.ListSubs(ctx, sub.FeedID, true)
	assert.Nil(t, err)
	assert.Empty(t, list)
	list, err = store.ListSubs(ctx, sub.FeedID, false)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Sub{sub}, list)
	stored, err = store.NextSub(ctx, sub.FeedID)
	assert.Equal(t, feed.ErrNotFound, err)
	stored, err = store.GetSub(ctx, sub.SubID)
	assert.Nil(t, err)
	assert.Equal(t, sub, stored)
	cleared, err := store.DeleteSubs(ctx, sub.FeedID, "%nontest%")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), cleared)
	cleared, err = store.DeleteSubs(ctx, sub.FeedID, "%test%")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), cleared)
	stored, err = store.GetSub(ctx, sub.SubID)
	assert.Equal(t, feed.ErrNotFound, err)
	stored, err = store.NextSub(ctx, sub.FeedID)
	assert.Equal(t, feed.ErrNotFound, err)
}
