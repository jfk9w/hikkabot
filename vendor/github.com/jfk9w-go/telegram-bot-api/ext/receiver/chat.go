package receiver

import (
	"context"
	"strings"

	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/telegram-bot-api"
)

type Chat struct {
	Sender    telegram.Sender
	ID        telegram.ChatID
	Silent    bool
	Preview   bool
	ParseMode telegram.ParseMode
}

func (r *Chat) String() string {
	return "telegram.chat." + r.ID.String()
}

func (r *Chat) SendText(ctx context.Context, text string) error {
	return r.sendText(ctx, text, r.Preview)
}

func (r *Chat) SendMedia(ctx context.Context, ref MediaRef, caption string) error {
	media, err := ref.Get(ctx)
	if err == nil {
		if media == nil {
			return nil
		}

		payload := &telegram.Media{
			Type:      telegram.MediaTypeByMIMEType(media.MIMEType),
			Input:     media.Input,
			Caption:   caption,
			ParseMode: r.ParseMode,
		}

		_, err = r.Sender.Send(ctx, r.ID, payload, &telegram.SendOptions{
			DisableNotification: r.Silent,
			ReplyMarkup:         replyMarkup(ctx),
		})

		logf.Get(r).Resultf(ctx, logf.Debug, logf.Warn, "send media [%s]: %v", media.MIMEType, err)
		if err == nil {
			return nil
		}
	} else if isSkipOnMediaError(ctx) {
		logf.Get(r).Warnf(ctx, "send media failed (skipping): %v", err)
		return nil
	}

	return r.sendText(ctx, caption, true)
}

func (r *Chat) sendText(ctx context.Context, text string, preview bool) error {
	if text == "" {
		return nil
	}

	payload := &telegram.Text{
		Text:                  text,
		ParseMode:             r.ParseMode,
		DisableWebPagePreview: !preview,
	}

	_, err := r.Sender.Send(ctx, r.ID, payload, &telegram.SendOptions{
		DisableNotification: r.Silent,
		ReplyMarkup:         replyMarkup(ctx),
	})

	logf.Get(r).Resultf(ctx, logf.Debug, logf.Warn, "send text [%s]: %v", cut(text, 50), err)
	return err
}

func cut(value string, size int) string {
	if len(value) < size {
		return value
	}

	newLine := strings.Index(value, "\n")
	if newLine >= 0 && newLine < size {
		size = newLine
	}

	return value[:size] + "..."
}
