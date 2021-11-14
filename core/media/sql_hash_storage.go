package media

import (
	"context"

	"github.com/jfk9w-go/flu/gormf"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SQLHashStorage gorm.DB

func (s *SQLHashStorage) Unmask() *gorm.DB {
	return (*gorm.DB)(s)
}

func (s *SQLHashStorage) Init(ctx context.Context) error {
	return s.Unmask().WithContext(ctx).AutoMigrate(new(Hash))
}

func (s *SQLHashStorage) Check(ctx context.Context, hash *Hash) (bool, error) {
	update := clause.Set{
		clause.Assignment{Column: clause.Column{Name: "collisions"}, Value: gorm.Expr("blob.collisions + 1")},
		clause.Assignment{Column: clause.Column{Name: "url"}, Value: hash.URL},
		clause.Assignment{Column: clause.Column{Name: "hash_type"}, Value: hash.Type},
		clause.Assignment{Column: clause.Column{Name: "hash"}, Value: hash.Value},
		clause.Assignment{Column: clause.Column{Name: "last_seen"}, Value: hash.LastSeen},
	}

	err := s.Unmask().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(gormf.OnConflictClause(hash, "primaryKey", false, update)).
			Create(hash).
			Error; err != nil {
			return errors.Wrap(err, "create")
		}

		if err := tx.First(hash).Error; err != nil {
			return errors.Wrap(err, "find")
		}

		return nil
	})

	ok := false
	if err == nil && hash.Collisions == 0 {
		ok = true
	}

	return ok, err
}
