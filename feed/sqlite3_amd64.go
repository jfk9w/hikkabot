package feed

import (
	"context"

	"github.com/doug-martin/goqu/v9"
)

func (s *SQLite3) selectSubs(ctx context.Context, sd *goqu.SelectDataset) ([]Sub, error) {
	subs := make([]Sub, 0)
	return subs, sd.Select(Sub{}).ScanStructsContext(ctx, &subs)
}

func insertSub(id *goqu.InsertDataset, sub Sub) *goqu.InsertDataset {
	return id.Rows(sub)
}
