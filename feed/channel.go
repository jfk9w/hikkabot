package feed

import (
	"log"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/pkg/errors"
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
	parseMode := update.Text.ParseMode
	pages := update.Text.Pages
	if parseMode != telegram.HTML {
		panic(errors.Errorf("unsupported parse mode: %s", parseMode))
	}
	if len(update.Media) == 1 && len(pages) == 1 {
		mediaFuture := update.Media[0]
		mediaURL := format.PrintHTMLLink("[media]", mediaFuture.URL)
		caption := mediaURL + "\n" + pages[0]
		if utf8.RuneCountInString(caption) <= telegram.MaxCaptionSize {
			media, err := mediaFuture.Result()
			if err == nil {
				media.Caption = caption
				media.ParseMode = parseMode
				_, err = tg.Send(chatID, media, DefaultSendUpdateOptions)
			}
			if err != nil {
				log.Printf("Failed to process media %s: %s", mediaFuture.URL, err)
				_, err = tg.Send(chatID,
					&telegram.Text{
						Text:      caption,
						ParseMode: parseMode},
					DefaultSendUpdateOptions)
			}
			return err
		}
	}

	for _, page := range pages {
		_, err := tg.Send(chatID,
			&telegram.Text{
				Text:                  page,
				ParseMode:             parseMode,
				DisableWebPagePreview: true},
			DefaultSendUpdateOptions)
		if err != nil {
			log.Printf("Failed to send message: %v. Message:\n%s", err, page)
			return err
		}
	}

	for _, mediaFuture := range update.Media {
		url := format.PrintHTMLLink("[media]", mediaFuture.URL)
		media, err := mediaFuture.Result()
		if err == nil {
			_, err = tg.Send(chatID, media, DefaultSendUpdateOptions)
		}
		if err != nil {
			log.Printf("Failed to process media %s: %s", mediaFuture.URL, err)
			_, err = tg.Send(chatID,
				&telegram.Text{
					Text:      url,
					ParseMode: parseMode},
				DefaultSendUpdateOptions)
		}
		if err != nil {
			return err
		}
	}

	return nil
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
