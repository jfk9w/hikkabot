package output

import "context"

type contextValues struct {
	pageSize int
	maxPages int
}

func With(ctx context.Context, pageSize int, maxPages int) context.Context {
	return context.WithValue(ctx, contextValues{}, contextValues{
		pageSize: pageSize,
		maxPages: maxPages,
	})
}

func values(ctx context.Context) (contextValues, bool) {
	values, ok := ctx.Value(contextValues{}).(contextValues)
	return values, ok
}

func pageSize(ctx context.Context) (int, bool) {
	values, ok := values(ctx)
	return values.pageSize, ok
}

func maxPages(ctx context.Context) (int, bool) {
	values, ok := values(ctx)
	return values.maxPages, ok
}
