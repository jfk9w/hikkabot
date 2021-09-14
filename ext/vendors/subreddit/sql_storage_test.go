package subreddit_test

import (
	"context"
	"testing"
	"time"

	"github.com/jfk9w-go/flu"
	gormutil "github.com/jfk9w-go/flu/gorm"
	"github.com/stretchr/testify/assert"

	"github.com/jfk9w/hikkabot/3rdparty/reddit"
	"github.com/jfk9w/hikkabot/ext/vendors/subreddit"
	"github.com/jfk9w/hikkabot/util"
)

func TestSQLStorage_Things(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	db := gormutil.NewTestPostgres(t)
	defer flu.CloseQuietly(db)

	storage := (*subreddit.SQLStorage)(db.DB)
	assert.Nil(t, storage.Init(ctx))

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	assert.Nil(t, storage.SaveThings(ctx, []reddit.Thing{{
		Data: reddit.ThingData{
			ID:        "1",
			CreatedAt: now,
			Subreddit: "test",
			Domain:    "test.com",
			Ups:       10,
			Author:    "test",
		},
		LastSeen: now,
	}}))

	thing := new(reddit.Thing)
	err = storage.Unmask().WithContext(ctx).
		First(thing).
		Error
	assert.Nil(t, err)
	assert.Equal(t, "1", thing.Data.ID)
	assert.Equal(t, 10, thing.Data.Ups)
	assert.Equal(t, now.UnixMilli(), thing.Data.CreatedAt.UnixMilli())
	assert.Equal(t, now.UnixMilli(), thing.LastSeen.UnixMilli())

	now = now.Add(time.Hour)
	assert.Nil(t, storage.SaveThings(ctx, []reddit.Thing{{
		Data: reddit.ThingData{
			ID:        "1",
			CreatedAt: now.Add(-time.Hour),
			Subreddit: "test",
			Domain:    "test.com",
			Ups:       15,
			Author:    "test",
		},
		LastSeen: now,
	}}))

	thing = new(reddit.Thing)
	err = storage.Unmask().WithContext(ctx).
		First(thing).
		Error
	assert.Nil(t, err)
	assert.Equal(t, 15, thing.Data.Ups)
	assert.Equal(t, now.Add(-time.Hour).UnixMilli(), thing.Data.CreatedAt.UnixMilli())
	assert.Equal(t, now.UnixMilli(), thing.LastSeen.UnixMilli())

	now = now.Add(time.Hour)
	assert.Nil(t, storage.SaveThings(ctx, []reddit.Thing{
		{
			Data: reddit.ThingData{
				ID:        "1",
				CreatedAt: now.Add(-2 * time.Hour),
				Subreddit: "test",
				Domain:    "test.com",
				Ups:       15,
				Author:    "test",
			},
			LastSeen: now,
		},
		{
			Data: reddit.ThingData{
				ID:        "2",
				CreatedAt: now,
				Subreddit: "test",
				Domain:    "test.com",
				Ups:       30,
				Author:    "test",
			},
		},
	}))

	percentile, err := storage.GetPercentile(ctx, "test", 0.51)
	assert.Nil(t, err)
	assert.Equal(t, 15, percentile)
	percentile, err = storage.GetPercentile(ctx, "test", 0.5)
	assert.Nil(t, err)
	assert.Equal(t, 30, percentile)

	deleted, err := storage.DeleteStaleThings(ctx, now.Add(-30*time.Minute))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), deleted)

	var count int64
	err = storage.Unmask().WithContext(ctx).
		Find(new(reddit.Thing)).
		Count(&count).
		Error
	assert.Nil(t, err)
	assert.Equal(t, int64(1), count)

	sentIDs := make(util.StringSet)
	sentIDs.Add("1")
	sentIDs.Add("2")
	freshSentIDs, err := storage.GetFreshThingIDs(ctx, sentIDs)
	assert.Nil(t, err)
	assert.Equal(t, []string{"1"}, freshSentIDs.Slice())
}

func getContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}
