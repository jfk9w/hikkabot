package ext

import (
	"context"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/jfk9w-go/telegram-bot-api/ext/output"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
)

func HTML(ctx context.Context, sender telegram.Sender, chatID telegram.ID) *html.Writer {
	return (&html.Writer{
		Out: &output.Paged{
			Receiver: &receiver.Chat{
				Sender:    sender,
				ID:        chatID,
				ParseMode: telegram.HTML,
			},
		},
	}).WithContext(output.With(ctx, telegram.MaxMessageSize, 0))
}
