package storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/ext/richtext"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/feed/storage"
	gormutil "github.com/jfk9w/hikkabot/util/gorm"
	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	null "gopkg.in/guregu/null.v3"
	"gorm.io/gorm"
)

func TestSQL_Feed(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	storage := createStorage(ctx, t)
	defer storage.Close()

	activeSubIDs, err := storage.Init(ctx)
	assert.Nil(t, err)
	assert.Empty(t, activeSubIDs)

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	sub := &feed.Subscription{
		Header: &feed.Header{
			SubID:  "123",
			Vendor: "test",
			FeedID: 456,
		},
		Name: "test subscription",
		Data: gormutil.Jsonb("{}"),
	}

	subs, err := storage.List(ctx, sub.Header.FeedID, true)
	assert.Nil(t, err)
	assert.Empty(t, subs)

	subs, err = storage.List(ctx, sub.Header.FeedID, false)
	assert.Nil(t, err)
	assert.Empty(t, subs)

	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, sub.Header, nil))
	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, sub.Header, errors.New("test error")))
	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, sub.Header, sub.Data))

	_, err = storage.Shift(ctx, sub.Header.FeedID)
	assert.Equal(t, feed.ErrNotFound, err)

	assert.Nil(t, storage.Create(ctx, sub))
	assert.Equal(t, feed.ErrExists, storage.Create(ctx, sub))

	activeSubIDs, err = storage.Init(ctx)
	assert.Nil(t, err)
	assert.Equal(t, []telegram.ID{sub.Header.FeedID}, activeSubIDs)

	subs, err = storage.List(ctx, sub.Header.FeedID, true)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Subscription{*sub}, subs)

	subs, err = storage.List(ctx, sub.Header.FeedID, false)
	assert.Nil(t, err)
	assert.Empty(t, subs)

	newSub, err := storage.Get(ctx, sub.Header)
	assert.Nil(t, err)
	assert.Equal(t, sub, newSub)

	newSub, err = storage.Shift(ctx, sub.Header.FeedID)
	assert.Nil(t, err)
	assert.Equal(t, sub, newSub)

	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, sub.Header, nil))

	sub.Error = null.StringFrom("test error")
	sub.UpdatedAt = &now
	assert.Nil(t, storage.Update(ctx, now, sub.Header, errors.New(sub.Error.String)))

	activeSubIDs, err = storage.Init(ctx)
	assert.Nil(t, err)
	assert.Empty(t, activeSubIDs)

	subs, err = storage.List(ctx, sub.Header.FeedID, false)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Subscription{*sub}, subs)

	newSub, err = storage.Get(ctx, sub.Header)
	assert.Nil(t, err)
	assert.Equal(t, sub, newSub)

	_, err = storage.Shift(ctx, sub.Header.FeedID)
	assert.Equal(t, feed.ErrNotFound, err)

	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, sub.Header, errors.New("test error")))
	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, sub.Header, gormutil.Jsonb(`{"x": "1"}`)))

	now = now.Add(time.Hour)
	sub.UpdatedAt = &now
	sub.Error = null.NewString("", false)
	assert.Nil(t, storage.Update(ctx, now, sub.Header, nil))

	activeSubIDs, err = storage.Init(ctx)
	assert.Nil(t, err)
	assert.Equal(t, []telegram.ID{sub.Header.FeedID}, activeSubIDs)

	subs, err = storage.List(ctx, sub.Header.FeedID, true)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Subscription{*sub}, subs)

	newSub, err = storage.Get(ctx, sub.Header)
	assert.Nil(t, err)
	assert.Equal(t, sub, newSub)

	newSub, err = storage.Shift(ctx, sub.Header.FeedID)
	assert.Nil(t, err)
	assert.Equal(t, sub, newSub)

	now = now.Add(time.Hour)
	sub.UpdatedAt = &now
	sub.Data = gormutil.Jsonb(`{"x": "1"}`)
	assert.Equal(t, nil, storage.Update(ctx, now, sub.Header, sub.Data))

	newSub, err = storage.Get(ctx, sub.Header)
	assert.Nil(t, err)
	assert.Equal(t, sub, newSub)

	assert.Nil(t, storage.Delete(ctx, sub.Header))

	_, err = storage.Get(ctx, sub.Header)
	assert.Equal(t, feed.ErrNotFound, err)
}

func TestSQL_BlobHash(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	storage := createStorage(ctx, t)
	defer storage.Close()

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	hash := &feed.BlobHash{
		FeedID:    456,
		URL:       "https://reddit.com",
		Type:      "md5",
		Hash:      "123",
		FirstSeen: now,
		LastSeen:  now,
	}

	assert.Nil(t, storage.Check(ctx, hash))

	newHash := new(feed.BlobHash)
	err = storage.Unmask().WithContext(ctx).
		First(newHash).
		Error
	assert.Nil(t, err)
	assert.Equal(t, hash, newHash)

	now = now.Add(time.Hour)
	hash.FirstSeen = now
	hash.LastSeen = now
	hash.URL = "https://google.com"
	assert.Equal(t, richtext.ErrSkipMedia, storage.Check(ctx, hash))

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

type testStorage struct {
	*storage.SQL
	Close func() error
}

func createStorage(ctx context.Context, t *testing.T) *testStorage {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatal(err)
	}

	container, err := pool.Run("postgres", "latest", []string{"POSTGRES_PASSWORD=pwd"})
	if err != nil {
		t.Fatal(err)
	}

	var db *gorm.DB
	dsn := fmt.Sprintf("postgresql://postgres:pwd@localhost:%s/postgres", container.GetPort("5432/tcp"))
	if err := pool.Retry(func() error {
		db, err = gormutil.NewPostgres(dsn)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	storage := (*storage.SQL)(db)
	_, err = storage.Init(ctx)
	assert.Nil(t, err)

	closer := func() error {
		db, err := db.DB()
		if err != nil {
			return err
		}

		if err := db.Close(); err != nil {
			return err
		}

		return container.Close()
	}

	return &testStorage{storage, closer}
}
