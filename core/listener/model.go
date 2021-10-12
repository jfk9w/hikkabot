package listener

import (
	"context"

	"github.com/jfk9w-go/telegram-bot-api"

	"hikkabot/core/feed"
)

type Vendor interface {
	OnCommand(ctx context.Context, client telegram.Client, cmd *telegram.Command) (bool, error)
}

type Aggregator interface {
	Subscribe(ctx context.Context, feedID telegram.ID, ref string, options []string) error
	Suspend(ctx context.Context, header *feed.Header, err error) error
	Resume(ctx context.Context, header *feed.Header) error
	Delete(ctx context.Context, header *feed.Header) error
	Clear(ctx context.Context, feedID telegram.ID, pattern string) error
	List(ctx context.Context, feedID telegram.ID, active bool) ([]feed.Subscription, error)
}

type AccessControl interface {
	GetChatLink(ctx context.Context, client telegram.Client, chatID telegram.ID) (string, error)
	CheckAccess(ctx context.Context, client telegram.Client, userID, chatID telegram.ID) (context.Context, error)
	NotifyAdmins(ctx context.Context, client telegram.Client,
		chatID telegram.ID, markup telegram.ReplyMarkup, writeHTML feed.WriteHTML) error
}
