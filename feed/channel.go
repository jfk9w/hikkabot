package feed

import (
	"log"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	_media "github.com/jfk9w/hikkabot/media"
)

type Channel interface {
	SendUpdate(telegram.ID, Update) error
	GetChat(telegram.ChatID) (*telegram.Chat, error)
	GetChatAdministrators(telegram.ChatID) ([]telegram.ChatMember, error)
	SendAlert([]telegram.ID, format.Text, telegram.ReplyMarkup)
}

type Telegram struct {
	telegram.Client
}

var DefaultSendUpdateOptions = &telegram.SendOptions{DisableNotification: true}

func (tg Telegram) SendUpdate(chatID telegram.ID, update Update) error {
	if len(update.Media) == 1 &&
		len(update.Pages) == 1 &&
		utf8.RuneCountInString(update.Pages[0])+len(update.Media[0].URL)+23 <= telegram.MaxCaptionSize {
		return tg.SendMedia(chatID, update.Media[0], update.Pages[0])
	}

	for _, page := range update.Pages {
		if _, err := tg.Send(chatID,
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
		if err := tg.SendMedia(chatID, promise, ""); err != nil {
			return err
		}
	}

	return nil
}

func (tg Telegram) SendMedia(chatID telegram.ID, promise *_media.Promise, text string) error {
	materialized, err := promise.Materialize()
	if err == _media.ErrFiltered {
		if text != "" {
			_, err = tg.Send(chatID,
				&telegram.Text{
					Text:                  text,
					ParseMode:             telegram.HTML,
					DisableWebPagePreview: true,
				},
				DefaultSendUpdateOptions)
			return err
		}

		return nil
	}

	caption := format.PrintHTMLLink("[media]", promise.URL)
	if text != "" {
		caption += "\n" + text
	}

	if err == nil {
		media := &telegram.Media{
			Type:      materialized.Type,
			Resource:  materialized.Resource,
			Caption:   caption,
			ParseMode: telegram.HTML,
		}

		_, err = tg.Send(chatID, media, DefaultSendUpdateOptions)
		materialized.Resource.Cleanup()
		if err == nil {
			return nil
		} else {
			log.Printf("Failed to send media %s: %s", promise.URL, err.Error())
		}
	}

	_, err = tg.Send(chatID, &telegram.Text{
		Text:      caption,
		ParseMode: telegram.HTML,
	}, nil)
	return err
}

func (tg Telegram) GetChat(chatID telegram.ChatID) (*telegram.Chat, error) {
	return tg.Client.GetChat(chatID)
}

func (tg Telegram) GetChatAdministrators(chatID telegram.ChatID) ([]telegram.ChatMember, error) {
	return tg.Client.GetChatAdministrators(chatID)
}

func (tg Telegram) SendAlert(chatIDs []telegram.ID, text format.Text, replyMarkup telegram.ReplyMarkup) {
	sendable := &telegram.Text{Text: text.Pages[0], ParseMode: text.ParseMode}
	options := &telegram.SendOptions{ReplyMarkup: replyMarkup}
	for _, chatID := range chatIDs {
		_, err := tg.Send(chatID, sendable, options)
		if err != nil {
			log.Printf("Failed to send alert to %d: %s", chatID, err)
		}
	}
}
