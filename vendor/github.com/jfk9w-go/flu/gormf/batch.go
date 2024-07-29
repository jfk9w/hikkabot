package gormf

import (
	"time"

	"github.com/jfk9w-go/flu/colf"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Batch is a slice of values.
type Batch[T any] []T

// EnsureSince detects time column (which must be tagged with `gorm:"time"`), removes all existing rows
// with time >= since and inserts the batch with Ensure.
func (b Batch[T]) EnsureSince(db *gorm.DB, since time.Time, keySetting string) error {
	timeColumns := CollectTaggedColumns([]T(b), "time")
	var timeColumn string
	for column := range timeColumns {
		timeColumn = column
		break
	}

	var model T
	if timeColumn == "" {
		return errors.Errorf("ensure since called, but no time fields found on %T", model)
	}

	tx, err := Filter(db, []T(b), "time", ">=", since)
	if err != nil {
		return err
	}

	if err := tx.Where(timeColumn+" >= ?", since).
		Delete(&model).
		Error; err != nil {
		return errors.Wrap(err, "drop")
	}

	return b.Ensure(db, keySetting)
}

// Ensure inserts values from this batch into db, updating existing rows.
// keySetting is used to specify struct field tag for collecting fields for ON CONFLICT clause.
func (b Batch[T]) Ensure(db *gorm.DB, keySetting string) error {
	onConflict := OnConflictClause([]T(b), keySetting, true, nil)
	return db.Clauses(onConflict).
		CreateInBatches([]T(b), 100).
		Error
}

// Filter applies filter based on tagged field struct.
// setting is used to specify gorm setting in struct field tag which is used to resolve the column for filtering.
func Filter(tx *gorm.DB, entity any, setting string, operator string, value any) (*gorm.DB, error) {
	columns := CollectTaggedColumns(entity, setting)
	if len(columns) > 1 {
		return nil, errors.Errorf("multiple fields have %s gorm setting: %v", setting, colf.Keys[string, string](columns))
	}

	for column := range columns {
		return tx.Where(column+" "+operator+" ?", value), nil
	}

	return tx, errors.Errorf("no fields with gorm setting %s found", setting)
}
