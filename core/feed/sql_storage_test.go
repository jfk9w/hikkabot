package feed_test

import (
	"context"
	"testing"
	"time"

	"github.com/jfk9w-go/flu/gorm"

	"github.com/jfk9w-go/flu"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	null "gopkg.in/guregu/null.v3"

	"github.com/jfk9w/hikkabot/core/feed"
)

func TestSQLStorage(t *testing.T) {
	ctx, cancel := getContext()
	defer cancel()

	db := gorm.NewTestPostgres(t)
	defer flu.CloseQuietly(db)

	storage := (*feed.SQLStorage)(db.DB)
	assert.Nil(t, storage.Init(ctx))

	activeSubIDs, err := storage.Active(ctx)
	assert.Nil(t, err)
	assert.Empty(t, activeSubIDs)

	now, err := time.Parse(time.RFC3339, "2021-07-28T03:00:00+03:00")
	assert.Nil(t, err)

	header := &feed.Header{
		SubID:  "123",
		Vendor: "test",
		FeedID: 456,
	}

	sub := &feed.Subscription{
		Header: header,
		Name:   "test subscription",
		Data:   gorm.JSONB("{}"),
	}

	subs, err := storage.List(ctx, header.FeedID, true)
	assert.Nil(t, err)
	assert.Empty(t, subs)

	subs, err = storage.List(ctx, header.FeedID, false)
	assert.Nil(t, err)
	assert.Empty(t, subs)

	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, header, nil))
	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, header, errors.New("test error")))
	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, header, gorm.JSONB("{}")))

	_, err = storage.Shift(ctx, header.FeedID)
	assert.Equal(t, feed.ErrNotFound, err)

	assert.Nil(t, storage.Create(ctx, sub))
	assert.Equal(t, feed.ErrExists, storage.Create(ctx, sub))

	activeSubIDs, err = storage.Active(ctx)
	assert.Nil(t, err)
	assert.Equal(t, []telegram.ID{header.FeedID}, activeSubIDs)

	subs, err = storage.List(ctx, header.FeedID, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, header, subs[0].Header)
	assert.Equal(t, "test subscription", subs[0].Name)
	assert.Equal(t, gorm.JSONB("{}"), subs[0].Data)
	assert.Equal(t, (*time.Time)(nil), subs[0].UpdatedAt)
	assert.Equal(t, null.NewString("", false), subs[0].Error)

	subs, err = storage.List(ctx, header.FeedID, false)
	assert.Nil(t, err)
	assert.Empty(t, subs)

	sub, err = storage.Get(ctx, header)
	assert.Nil(t, err)
	assert.Equal(t, header, sub.Header)
	assert.Equal(t, "test subscription", sub.Name)
	assert.Equal(t, gorm.JSONB("{}"), sub.Data)
	assert.Equal(t, (*time.Time)(nil), sub.UpdatedAt)
	assert.Equal(t, null.NewString("", false), sub.Error)

	sub, err = storage.Shift(ctx, header.FeedID)
	assert.Nil(t, err)
	assert.Equal(t, header, sub.Header)

	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, header, nil))

	assert.Nil(t, storage.Update(ctx, now, header, errors.New("lol error")))

	activeSubIDs, err = storage.Active(ctx)
	assert.Nil(t, err)
	assert.Empty(t, activeSubIDs)

	subs, err = storage.List(ctx, header.FeedID, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, header, subs[0].Header)
	assert.Equal(t, now.UnixMilli(), subs[0].UpdatedAt.UnixMilli())
	assert.Equal(t, null.StringFrom("lol error"), subs[0].Error)

	sub, err = storage.Get(ctx, header)
	assert.Nil(t, err)
	assert.Equal(t, header, sub.Header)
	assert.Equal(t, null.StringFrom("lol error"), sub.Error)

	_, err = storage.Shift(ctx, header.FeedID)
	assert.Equal(t, feed.ErrNotFound, err)

	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, header, errors.New("test error")))
	assert.Equal(t, feed.ErrNotFound, storage.Update(ctx, now, header, gorm.JSONB(`{"x": "1"}`)))

	now = now.Add(time.Hour)
	sub.UpdatedAt = &now
	sub.Error = null.NewString("", false)
	assert.Nil(t, storage.Update(ctx, now, header, nil))

	activeSubIDs, err = storage.Active(ctx)
	assert.Nil(t, err)
	assert.Equal(t, []telegram.ID{header.FeedID}, activeSubIDs)

	subs, err = storage.List(ctx, header.FeedID, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, header, subs[0].Header)
	assert.Equal(t, null.NewString("", false), subs[0].Error)

	sub, err = storage.Get(ctx, header)
	assert.Nil(t, err)
	assert.Equal(t, null.NewString("", false), sub.Error)

	sub, err = storage.Shift(ctx, header.FeedID)
	assert.Nil(t, err)
	assert.Equal(t, header, sub.Header)

	now = now.Add(time.Hour)
	sub.UpdatedAt = &now
	assert.Equal(t, nil, storage.Update(ctx, now, header, gorm.JSONB(`{"x": "1"}`)))

	sub, err = storage.Get(ctx, header)
	assert.Nil(t, err)
	assert.Equal(t, gorm.JSONB(`{"x": "1"}`), sub.Data)
	assert.Equal(t, now.UnixMilli(), sub.UpdatedAt.UnixMilli())

	assert.Nil(t, storage.Delete(ctx, sub.Header))

	_, err = storage.Get(ctx, sub.Header)
	assert.Equal(t, feed.ErrNotFound, err)
}

func getContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), time.Minute)
}
