package core

import (
	"context"

	"hikkabot/core/internal/storage"
	"hikkabot/feed"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/pkg/errors"
)

type StorageService interface {
	feed.Storage
	feed.EventStorage
	feed.MediaHashStorage
}

type StorageContext interface {
	StorageConfig() apfel.GormConfig
}

type Storage[C StorageContext] struct {
	StorageService
}

func (s Storage[C]) String() string {
	return storage.ServiceID
}

func (s *Storage[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	if s.StorageService != nil {
		return nil
	}

	db := &apfel.GormDB[C]{Config: app.Config().StorageConfig()}
	if err := app.Use(ctx, db, false); err != nil {
		return err
	}

	if err := db.DB().AutoMigrate(new(feed.Subscription), new(feed.Event), new(feed.MediaHash)); err != nil {
		return errors.Wrap(err, "auto-migrate")
	}

	s.StorageService = &storage.SQL{
		Clock: app,
		DB:    db.DB(),
	}

	return nil
}
