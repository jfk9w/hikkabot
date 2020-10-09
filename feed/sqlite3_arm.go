package feed

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/pkg/errors"
)

var columnOrder = []interface{}{
	"sub_id",
	"vendor",
	"feed_id",
	"name",
	"data",
	"updated_at",
}

func (s *SQLite3) selectSubs(ctx context.Context, builder *goqu.SelectDataset) ([]Sub, error) {
	rows, err := s.QuerySQLBuilder(ctx, builder.Select(columnOrder...))
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}

	defer rows.Close()
	subs := make([]Sub, 0)
	for rows.Next() {
		sub := Sub{}
		if err := rows.Scan(
			&sub.SubID.ID,
			&sub.SubID.Vendor,
			&sub.SubID.FeedID,
			&sub.Name,
			&sub.Data,
			&sub.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "scan")
		}

		subs = append(subs, sub)
	}

	return subs, nil
}

func insertSub(id *goqu.InsertDataset, sub Sub) *goqu.InsertDataset {
	return id.
		Cols(columnOrder...).
		Vals([]interface{}{
			sub.SubID.ID,
			sub.SubID.Vendor,
			sub.SubID.FeedID,
			sub.Name,
			sub.Data,
			sub.UpdatedAt,
		})
}
