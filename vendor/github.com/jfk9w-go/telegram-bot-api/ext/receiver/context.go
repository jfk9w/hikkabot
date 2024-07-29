package receiver

import (
	"context"

	"github.com/jfk9w-go/telegram-bot-api"
)

type replyMarkupKey struct{}

func ReplyMarkup(ctx context.Context, replyMarkup telegram.ReplyMarkup) context.Context {
	return context.WithValue(ctx, replyMarkupKey{}, replyMarkup)
}

func replyMarkup(ctx context.Context) telegram.ReplyMarkup {
	value, _ := ctx.Value(replyMarkupKey{}).(telegram.ReplyMarkup)
	return value
}

type skipOnMediaError struct{}

func SkipOnMediaError(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipOnMediaError{}, true)
}

func isSkipOnMediaError(ctx context.Context) bool {
	_, ok := ctx.Value(skipOnMediaError{}).(bool)
	return ok
}
