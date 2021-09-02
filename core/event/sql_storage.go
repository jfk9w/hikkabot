package event

import (
	"context"

	"gorm.io/gorm"
)

type SQLStorage gorm.DB

func (s *SQLStorage) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQLStorage) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(Log))
}

func (s *SQLStorage) SaveEvent(ctx context.Context, row *Log) error {
	return s.Unmask().WithContext(ctx).Create(row).Error
}
