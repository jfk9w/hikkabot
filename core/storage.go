package core

import (
	"context"

	"hikkabot/core/internal/storage"
	"hikkabot/feed"

	"github.com/jfk9w-go/flu/logf"

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

	config := app.Config().StorageConfig()
	if config.Driver != "postgres" {
		logf.Get(s).Warnf(ctx, "database driver is not postgres – some functions will be unavailable; "+
			"consider switching to postgres")
	}

	db := &apfel.GormDB[C]{Config: config}
	if err := app.Use(ctx, db, false); err != nil {
		return err
	}

	if err := db.DB().AutoMigrate(new(feed.Subscription), new(feed.Event), new(feed.MediaHash)); err != nil {
		return errors.Wrap(err, "auto-migrate")
	}

	s.StorageService = &storage.SQL{
		Clock: app,
		DB:    db.DB().Debug(),
		IsPG:  db.Config.Driver == "postgres",
	}

	return nil
}
