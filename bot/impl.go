package bot

import (
	"fmt"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type T struct {
	Bot
	conv Converter
}

func Wrap(bot Bot, conv Converter) *T {
	return &T{bot, conv}
}

func (bot *T) SendText(chat telegram.ChatRef, text string, args ...interface{}) {
	text = fmt.Sprintf(text, args...)
	if err := <-bot.Send(telegram.SendMessageRequest{
		Chat: chat,
		Text: text,
	}, nil); err != nil {
		log.Warningf("Unable to send message to %s: %s", chat.String(), err)
	}
}

func (bot *T) NotifyAll(chats []telegram.ChatRef, text string, args ...interface{}) {
	text = fmt.Sprintf(text, args...)
	for _, chat := range chats {
		bot.SendText(chat, text)
	}
}

func (bot *T) GetAdmins(chat telegram.ChatRef) ([]telegram.ChatRef, error) {
	admins, err := bot.GetChatAdministrators(chat)
	if err != nil {
		if tgerr, ok := err.(telegram.Error); ok {
			if tgerr.Description == "Bad Request: there is no administrators in the private chat" {
				return []telegram.ChatRef{chat}, nil
			}
		}

		log.Warningf("Cannot get chat %s administrator list: %s", chat.String(), err)
		return nil, errors.New("unable to get chat admin list")
	}

	refs := make([]telegram.ChatRef, 0)
	for _, admin := range admins {
		if !admin.User.IsBot {
			refs = append(refs, admin.User.Ref())
		}
	}

	return refs, nil
}

func (bot *T) SendHtml(chat telegram.ChatRef, html string) error {
	if err := <-bot.Send(telegram.SendMessageRequest{
		Chat:      chat,
		ParseMode: telegram.HTML,
		Text:      html,
	}, nil); err != nil {
		log.Errorf("Failed to send html to %s: %s", chat, err)
		return err
	}

	return nil
}

func link(url string) string {
	return fmt.Sprintf(`<a href="%s">[A]</a>`, html.EscapeString(url))
}

func (bot *T) SendLink(chat telegram.ChatRef, url string) error {
	if err := <-bot.Send(telegram.SendMessageRequest{
		Chat:      chat,
		ParseMode: telegram.HTML,
		Text:      link(url),
	}, nil); err != nil {
		log.Errorf("Failed to send link %s to %s: %s", url, chat, err)
		return err
	}

	return nil
}

func (bot *T) SendFile(chat telegram.ChatRef, file *dvach.File) error {
	url := file.URL()
	if file.Type == dvach.Webm {
		mp4, err := bot.conv.Get(url)
		if err != nil {
			log.Warningf("Webm %s failed to convert: %s", file.URL(), err)
			return bot.SendLink(chat, url)
		} else {
			url = mp4
		}
	}

	mediaType := telegram.Photo
	if file.DurationSecs != nil {
		mediaType = telegram.Video
	}

	base := telegram.BaseInputMedia{
		Type0:      mediaType,
		Media0:     url,
		ParseMode0: telegram.HTML,
		Caption0:   link(url),
	}

	var media telegram.InputMedia
	if mediaType == telegram.Photo {
		media = telegram.InputMediaPhoto{base}
	} else {
		video := telegram.InputMediaVideo{
			BaseInputMedia: base,
		}

		if file.DurationSecs != nil {
			video.Duration = *file.DurationSecs
		}
		if file.Width != nil {
			video.Width = *file.Width
		}
		if file.Height != nil {
			video.Height = *file.Height
		}

		media = video
	}

	if err := <-bot.Send(telegram.SendMediaRequest{
		Chat:                chat,
		Media:               []telegram.InputMedia{media},
		DisableNotification: true,
	}, nil); err != nil {
		log.Warningf("Failed to send %s as media to %s: %s", url, chat, err)
		return bot.SendLink(chat, url)
	}

	return nil
}

func (bot *T) SendPost(chat telegram.ChatRef, post text.Post) error {
	parts := text.FormatPost(post)
	for _, part := range parts {
		if err := bot.SendHtml(chat, part); err != nil {
			return err
		}
	}

	files := post.Files
	for _, file := range files {
		if err := bot.SendFile(chat, file); err != nil {
			return err
		}
	}

	return nil
}

func (bot *T) SendPopular(chat telegram.ChatRef, threads []*dvach.Thread, searchText []string) error {
	parts := text.Search(threads, searchText)
	for _, part := range parts {
		if err := bot.SendHtml(chat, part); err != nil {
			return err
		}
	}

	if len(parts) == 0 {
		if err := bot.SendHtml(chat, "empty"); err != nil {
			return err
		}
	}

	return nil
}
