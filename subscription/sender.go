package subscription

import (
	"os"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/format"
	"github.com/jfk9w/hikkabot/media"
	"golang.org/x/exp/utf8string"
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
		len(update.Text) == 1 &&
		utf8string.NewString(update.Text[0]).RuneCount() < telegram.MaxCaptionSize
	if canCollapse {
		if len(update.Media) == 1 {
			return s.sendMedia(update.Media[0], update.Text[0])
		} else {
			return s.sendText(update.Text[0])
		}
	} else {
		for _, text := range update.Text {
			err := s.sendText(text)
			if err != nil {
				return err
			}
		}
		for i := range update.Media {
			err := s.sendMedia(update.Media[i], "")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Sender) sendMedia(media *media.Media, caption string) error {
	if caption != "" {
		caption = "<br>" + caption
	}
	caption = format.NewHTML(telegram.MaxCaptionSize, 1, nil, nil).
		Link("[media]", media.Href).
		Parse(caption).
		Pages()[0]
	file, mediaType, err := media.WaitForResult()
	if err == nil {
		defer os.RemoveAll(file.Path())
		_, err = s.bot.Send(s.chatID,
			&telegram.Media{
				Type:      mediaType.TelegramType(),
				Resource:  file,
				Caption:   caption,
				ParseMode: telegram.HTML},
			&telegram.SendOptions{DisableNotification: true})
	}
	if err != nil {
		_, err = s.bot.Send(s.chatID,
			&telegram.Text{
				Text:      caption,
				ParseMode: telegram.HTML},
			&telegram.SendOptions{DisableNotification: true})
	}
	return err
}

func (s *Sender) sendText(text string) error {
	if text == "" {
		return nil
	}
	_, err := s.bot.Send(s.chatID,
		&telegram.Text{
			Text:                  text,
			ParseMode:             telegram.HTML,
			DisableWebPagePreview: true},
		&telegram.SendOptions{DisableNotification: true})
	return err
}
