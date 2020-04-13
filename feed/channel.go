package feed

import (
	"context"
	"log"
	"time"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	_media "github.com/jfk9w/hikkabot/media"
)

type Channel interface {
	SendUpdate(context.Context, telegram.ID, Update) error
	GetChat(context.Context, telegram.ChatID) (*telegram.Chat, error)
	GetChatAdministrators(context.Context, telegram.ChatID) ([]telegram.ChatMember, error)
	SendAlert(context.Context, []telegram.ID, format.Text, telegram.ReplyMarkup)
}

type Telegram struct {
	telegram.Client
}

var DefaultSendUpdateOptions = &telegram.SendOptions{DisableNotification: true}

func (tg Telegram) SendUpdate(ctx context.Context, chatID telegram.ID, update Update) error {
	if len(update.Media) == 1 &&
		len(update.Pages) == 1 &&
		utf8.RuneCountInString(update.Pages[0])+len(update.Media[0].URL)+23 <= telegram.MaxCaptionSize {
		return tg.SendMedia(ctx, chatID, update.Media[0], update.Pages[0])
	}

	for _, page := range update.Pages {
		if _, err := tg.Send(ctx, chatID,
			&telegram.Text{
				Text:                  page,
				ParseMode:             telegram.HTML,
				DisableWebPagePreview: true,
			},
			DefaultSendUpdateOptions); err != nil {
			return err
		}
	}

	for _, promise := range update.Media {
		if err := tg.SendMedia(ctx, chatID, promise, ""); err != nil {
			return err
		}
	}

	return nil
}

func (tg Telegram) SendMedia(ctx context.Context, chatID telegram.ID, promise *_media.Promise, text string) error {
	materialized, err := promise.Materialize()
	if err == _media.ErrFiltered {
		return nil
	}

	caption := format.PrintHTMLLink("[media]", promise.URL)
	if text != "" {
		caption += "\n" + text
	}

	if err == nil {
		media := &telegram.Media{
			Type:      materialized.Type,
			Input:     materialized.Resource,
			Caption:   caption,
			ParseMode: telegram.HTML,
		}

		_, err = tg.Send(ctx, chatID, media, DefaultSendUpdateOptions)
		materialized.Resource.Cleanup()
		if err == nil {
			return nil
		}
	}

	if err != nil {
		log.Printf("Failed to send media %s as resource: %s", promise.URL, err)
	}

	_, err = tg.Send(ctx, chatID, &telegram.Text{
		Text:      caption,
		ParseMode: telegram.HTML,
	}, nil)
	return err
}

func (tg Telegram) GetChat(ctx context.Context, chatID telegram.ChatID) (*telegram.Chat, error) {
	return tg.Client.GetChat(ctx, chatID)
}

func (tg Telegram) GetChatAdministrators(ctx context.Context, chatID telegram.ChatID) ([]telegram.ChatMember, error) {
	return tg.Client.GetChatAdministrators(ctx, chatID)
}

func (tg Telegram) SendAlert(ctx context.Context, chatIDs []telegram.ID, text format.Text, replyMarkup telegram.ReplyMarkup) {
	sendable := &telegram.Text{Text: text.Pages[0], ParseMode: text.ParseMode}
	options := &telegram.SendOptions{ReplyMarkup: replyMarkup}
	for _, chatID := range chatIDs {
		for retry := 0; true; retry++ {
			if retry > 0 {
				time.Sleep(time.Duration(retry*retry) * time.Second)
			}
			if _, err := tg.Send(ctx, chatID, sendable, options); err != nil {
				if _, ok := err.(telegram.Error); !ok {
					log.Printf("Failed to send alert to %d, retrying: %s", chatID, err)
				} else {
					log.Printf("Failed to send alert to %d: %s", chatID, err)
					break
				}
			} else {
				break
			}
		}
	}
}
