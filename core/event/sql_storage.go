package event

import (
	"context"

	"github.com/jfk9w-go/telegram-bot-api"
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

func (s *SQLStorage) DeleteEvents(ctx context.Context, chatID, messageID, userID telegram.ID, types ...string) error {
	return s.Unmask().WithContext(ctx).
		Delete(new(Log),
			"chat_id = ? and message_id = ? and user_id = ? and type in ?",
			chatID, messageID, userID, types).
		Error
}

func (s *SQLStorage) CountEvents(ctx context.Context, chatID, messageID telegram.ID, types ...string) (map[string]int64, error) {
	rows := make([]struct {
		Type   string
		Events int64
	}, 0)

	if err := s.Unmask().WithContext(ctx).Raw( /* language=SQL */ `
		select type, count(1) as events
		from event
		where chat_id = ? and message_id = ? and type in ?
		group by type`,
		chatID, messageID, types).
		Scan(&rows).
		Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, row := range rows {
		stats[row.Type] = row.Events
	}

	return stats, nil
}
