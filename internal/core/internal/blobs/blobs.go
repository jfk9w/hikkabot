package blobs

import "context"

const ServiceID = "core.blobs"

type skipSizeCheckKey struct{}

func SkipSizeCheck(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipSizeCheckKey{}, true)
}

func skipSizeCheck(ctx context.Context) bool {
	_, ok := ctx.Value(skipSizeCheckKey{}).(bool)
	return ok
}
