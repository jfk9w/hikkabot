package subscription

import (
	"os"

	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/app/media"
	"github.com/jfk9w/hikkabot/html"
	"golang.org/x/exp/utf8string"
)

type Controller struct {
	Telegram *telegram.Bot
	Context  Context
	Services map[string]Service
}

func (c *Controller) sendUpdate(chatID telegram.ChatID, u Update) error {
	canCollapse := len(u.Media) <= 1 &&
		len(u.Text) == 1 &&
		utf8string.NewString(u.Text[0]).RuneCount() < MaxCollapsedCaptionSize

	if canCollapse {
		if len(u.Media) == 1 {
			return c.sendMedia(chatID, &u.Media[0], u.Text[0])
		} else {
			return c.sendText(chatID, u.Text[0])
		}
	} else {
		for _, p := range u.Text {
			err := c.sendText(chatID, p)
			if err != nil {
				return err
			}
		}

		for i := range u.Media {
			err := c.sendMedia(chatID, &u.Media[i], "")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) sendMedia(chatID telegram.ChatID, m *media.Media, caption string) error {
	if caption != "" {
		caption = "\n" + caption
	}

	caption = html.Link(m.Href, "[media]") + caption
	resource, mediaType, err := m.Get()
	if err == nil {
		defer os.RemoveAll(resource.Path())
		_, err = c.Telegram.Send(chatID,
			&telegram.Media{
				Type:      mediaType.TelegramType(),
				Resource:  resource,
				Caption:   caption,
				ParseMode: telegram.HTML},
			&telegram.SendOpts{DisableNotification: true})
	}

	if err != nil {
		_, err = c.Telegram.Send(chatID,
			&telegram.Text{
				Text:      caption,
				ParseMode: telegram.HTML},
			&telegram.SendOpts{DisableNotification: true})
	}

	return err
}

func (c *Controller) sendText(chatID telegram.ChatID, text string) error {
	if text == "" {
		return nil
	}

	_, err := c.Telegram.Send(chatID,
		&telegram.Text{
			Text:                  text,
			ParseMode:             telegram.HTML,
			DisableWebPagePreview: true},
		&telegram.SendOpts{DisableNotification: true})

	return err
}
