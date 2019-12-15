package subscription

import (
	"os"
	"unicode/utf8"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
)

type Sender struct {
	bot    telegram.Bot
	chatID telegram.ChatID
}

func NewSender(bot telegram.Bot, chatID telegram.ChatID) *Sender {
	return &Sender{bot, chatID}
}

func (s *Sender) Send(update Update) error {
	canCollapse := len(update.Media) <= 1 &&
		len(update.Text.Pages) == 1 &&
		utf8.RuneCountInString(update.Text.Pages[0]) < telegram.MaxCaptionSize
	if canCollapse {
		if len(update.Media) == 1 {
			return s.sendMedia(update.Media[0], update.Text.Pages[0], update.Text.ParseMode)
		} else {
			return s.sendText(update.Text.Pages[0], update.Text.ParseMode)
		}
	} else {
		for _, page := range update.Text.Pages {
			err := s.sendText(page, update.Text.ParseMode)
			if err != nil {
				return err
			}
		}
		for i := range update.Media {
			err := s.sendMedia(update.Media[i], "", update.Text.ParseMode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Sender) sendMedia(media *media.Media, caption string, parseMode telegram.ParseMode) error {
	if caption != "" {
		caption = "<br>" + caption
	}
	caption = format.NewHTML(telegram.MaxCaptionSize, 1, nil, nil).
		Link("[media]", media.Href).
		Parse(caption).
		Format().Pages[0]
	file, type_, err := media.WaitForResult()
	if err == nil {
		defer os.RemoveAll(file.Path())
		_, err = s.bot.Send(s.chatID,
			&telegram.Media{
				Type:      type_.TelegramType(),
				Resource:  file,
				Caption:   caption,
				ParseMode: parseMode},
			&telegram.SendOptions{DisableNotification: true})
	}
	if err != nil {
		_, err = s.bot.Send(s.chatID,
			&telegram.Text{
				Text:      caption,
				ParseMode: parseMode},
			&telegram.SendOptions{DisableNotification: true})
	}
	return err
}

func (s *Sender) sendText(text string, parseMode telegram.ParseMode) error {
	if text == "" {
		return nil
	}
	_, err := s.bot.Send(s.chatID,
		&telegram.Text{
			Text:                  text,
			ParseMode:             parseMode,
			DisableWebPagePreview: true},
		&telegram.SendOptions{DisableNotification: true})
	return err
}
