package media_test

import (
	"context"
	"testing"
	"time"

	media "github.com/jfk9w/hikkabot/core/media"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/stretchr/testify/assert"
)

func TestSQLHashStorage(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	db := gormutil.NewTestDatabase(t)
	defer db.Close()

	storage := (*media.SQLHashStorage)(db.DB)
	assert.Nil(t, storage.Init(ctx))

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	hash := &media.Hash{
		FeedID:    456,
		URL:       "https://reddit.com",
		Type:      "md5",
		Value:     "123",
		FirstSeen: now,
		LastSeen:  now,
	}

	ok, err := storage.Check(ctx, hash)
	assert.Nil(t, nil)
	assert.True(t, ok)

	newHash := new(media.Hash)
	err = storage.Unmask().WithContext(ctx).
		First(newHash).
		Error
	assert.Nil(t, err)
	assert.Equal(t, hash, newHash)

	now = now.Add(time.Hour)
	hash.FirstSeen = now
	hash.LastSeen = now
	hash.URL = "https://google.com"

	ok, err = storage.Check(ctx, hash)
	assert.Nil(t, err)
	assert.False(t, ok)

	err = storage.Unmask().WithContext(ctx).
		First(newHash).
		Error
	hash.FirstSeen = now.Add(-time.Hour)
	hash.Collisions = 1
	assert.Nil(t, err)
	assert.Equal(t, hash, newHash)
}

func getContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}
