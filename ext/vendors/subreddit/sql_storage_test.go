package subreddit_test

import (
	"context"
	"testing"
	"time"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/ext/vendors/subreddit"
	"github.com/jfk9w/hikkabot/util"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/stretchr/testify/assert"
)

func TestSQLStorage_Things(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	db := gormutil.NewTestDatabase(t)
	defer db.Close()

	storage := (*subreddit.SQLStorage)(db.DB)
	assert.Nil(t, storage.Init(ctx))

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	things := []reddit.Thing{{
		Data: reddit.ThingData{
			ID:        1,
			CreatedAt: now,
			Subreddit: "test",
			Domain:    "test.com",
			Ups:       10,
			Author:    "test",
		},
		LastSeen: now,
	}}

	assert.Nil(t, storage.SaveThings(ctx, things))

	newThing := new(reddit.Thing)
	err = storage.Unmask().WithContext(ctx).
		First(newThing).
		Error
	assert.Nil(t, err)
	assert.Equal(t, &things[0], newThing)

	now = now.Add(time.Hour)
	things[0].LastSeen = now
	things[0].Data.Ups = 15
	assert.Nil(t, storage.SaveThings(ctx, things))

	newThing = new(reddit.Thing)
	err = storage.Unmask().WithContext(ctx).
		First(newThing).
		Error
	assert.Nil(t, err)
	assert.Equal(t, &things[0], newThing)

	now = now.Add(time.Hour)
	things = append(things, things[0])
	things[1].Data.ID = 2
	things[1].Data.CreatedAt = now
	things[1].Data.Ups = 30
	things[1].LastSeen = now
	assert.Nil(t, storage.SaveThings(ctx, things))

	newThings := make([]reddit.Thing, 0)
	err = storage.Unmask().WithContext(ctx).
		Order("id asc").
		Find(&newThings).
		Error
	assert.Nil(t, err)
	assert.Equal(t, things, newThings)

	percentile, err := storage.GetPercentile(ctx, things[0].Data.Subreddit, 0.51)
	assert.Nil(t, err)
	assert.Equal(t, 15, percentile)
	percentile, err = storage.GetPercentile(ctx, things[0].Data.Subreddit, 0.5)
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
	assert.Equal(t, things[:1], things)

	sentIDs := make(util.Uint64Set)
	sentIDs.Add(newThing.Data.ID)
	sentIDs.Add(things[0].Data.ID)
	freshSentIDs, err := storage.GetFreshThingIDs(ctx, things[0].Data.Subreddit, sentIDs)
	assert.Nil(t, err)
	assert.Equal(t, []uint64{things[0].Data.ID}, freshSentIDs.Slice())
}

func getContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}
