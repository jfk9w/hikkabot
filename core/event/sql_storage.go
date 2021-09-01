package event

import (
	"context"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"gorm.io/gorm"
)

type SQLStorage gorm.DB

func (s *SQLStorage) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQLStorage) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(Log))
}

func (s *SQLStorage) IsKnownUser(ctx context.Context, userID telegram.ID) (bool, error) {
	var count int64
	return count > 0, s.Unmask().WithContext(ctx).
		Model(new(Log)).
		Where("user_id = ?", userID).
		Limit(1).
		Count(&count).
		Error
}

func (s *SQLStorage) SaveEvent(ctx context.Context, row *Log) error {
	return s.Unmask().WithContext(ctx).Create(row).Error
}
