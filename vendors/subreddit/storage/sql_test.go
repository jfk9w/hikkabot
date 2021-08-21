package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/util"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/jfk9w/hikkabot/vendors/subreddit/storage"
	"github.com/stretchr/testify/assert"
)

func TestSQL_Things(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	db := gormutil.NewTestDatabase(t)
	defer db.Close()
	storage := (*storage.SQL)(db.DB)

	err := storage.Init(ctx)
	assert.Nil(t, err)

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	thing := &reddit.ThingData{
		ID:        1,
		CreatedAt: now,
		Subreddit: "test",
		Domain:    "test.com",
		Ups:       10,
		Author:    "test",
		LastSeen:  now,
	}

	assert.Nil(t, storage.SaveThing(ctx, thing))

	newThing := new(reddit.ThingData)
	err = storage.Unmask().WithContext(ctx).
		First(newThing).
		Error
	assert.Nil(t, err)
	assert.Equal(t, thing, newThing)

	now = now.Add(time.Hour)
	thing.LastSeen = now
	thing.Ups = 15
	assert.Nil(t, storage.SaveThing(ctx, thing))

	newThing = new(reddit.ThingData)
	err = storage.Unmask().WithContext(ctx).
		First(newThing).
		Error
	assert.Nil(t, err)
	assert.Equal(t, thing, newThing)

	now = now.Add(time.Hour)
	thing.ID = 2
	thing.CreatedAt = now
	thing.Ups = 30
	thing.LastSeen = now
	assert.Nil(t, storage.SaveThing(ctx, thing))

	things := make([]reddit.ThingData, 2)
	err = storage.Unmask().WithContext(ctx).
		Order("id asc").
		Find(&things).
		Error
	assert.Nil(t, err)
	assert.Equal(t, []reddit.ThingData{*newThing, *thing}, things)

	percentile, err := storage.GetPercentile(ctx, thing.Subreddit, 0.51)
	assert.Nil(t, err)
	assert.Equal(t, 15, percentile)
	percentile, err = storage.GetPercentile(ctx, thing.Subreddit, 0.5)
	assert.Nil(t, err)
	assert.Equal(t, 30, percentile)

	deleted, err := storage.DeleteStaleThings(ctx, now.Add(-30*time.Minute))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), deleted)

	err = storage.Unmask().WithContext(ctx).
		Order("id asc").
		Find(&things).
		Error
	assert.Nil(t, err)
	assert.Equal(t, []reddit.ThingData{*thing}, things)

	sentIDs := make(util.Uint64Set)
	sentIDs.Add(newThing.ID)
	sentIDs.Add(thing.ID)
	freshSentIDs, err := storage.GetFreshThingIDs(ctx, thing.Subreddit, sentIDs)
	assert.Nil(t, err)
	assert.Equal(t, []uint64{thing.ID}, freshSentIDs.Slice())
}

func getContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}
