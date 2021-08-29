package media_test

import (
	"context"
	"testing"
	"time"

	"github.com/jfk9w-go/flu"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	media "github.com/jfk9w/hikkabot/core/media"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/stretchr/testify/assert"
)

func TestSQLHashStorage(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	db := gormutil.NewTestDatabase(t)
	defer flu.CloseQuietly(db)

	storage := (*media.SQLHashStorage)(db.DB)
	assert.Nil(t, storage.Init(ctx))

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	ok, err := storage.Check(ctx, &media.Hash{
		FeedID:    456,
		URL:       "https://reddit.com",
		Type:      "md5",
		Value:     "123",
		FirstSeen: now,
		LastSeen:  now,
	})

	assert.Nil(t, err)
	assert.True(t, ok)

	hash := new(media.Hash)
	err = storage.Unmask().WithContext(ctx).
		First(hash).
		Error
	assert.Nil(t, err)
	assert.Equal(t, telegram.ID(456), hash.FeedID)
	assert.Equal(t, "https://reddit.com", hash.URL)
	assert.Equal(t, "md5", hash.Type)
	assert.Equal(t, "123", hash.Value)
	assert.Equal(t, int64(0), hash.Collisions)

	now = now.Add(time.Hour)
	ok, err = storage.Check(ctx, &media.Hash{
		FeedID:    456,
		URL:       "https://google.com",
		Type:      "md5",
		Value:     "123",
		FirstSeen: now,
		LastSeen:  now,
	})

	assert.Nil(t, err)
	assert.False(t, ok)

	err = storage.Unmask().WithContext(ctx).
		First(hash).
		Error
	assert.Nil(t, err)
	assert.Equal(t, "https://google.com", hash.URL)
	assert.Equal(t, now.Add(-time.Hour).UnixMilli(), hash.FirstSeen.UnixMilli())
	assert.Equal(t, now.UnixMilli(), hash.LastSeen.UnixMilli())
	assert.Equal(t, int64(1), hash.Collisions)
}

func getContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}
