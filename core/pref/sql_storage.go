package pref

import (
	"context"

	"gorm.io/gorm"
)

type SQLStorage gorm.DB

func (s *SQLStorage) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQLStorage) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(Interaction))
}

func (s *SQLStorage) SaveInteraction(ctx context.Context, interaction *Interaction) (likes int64, dislikes int64, err error) {
	err = s.Unmask().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return nil
	})

	return
}
