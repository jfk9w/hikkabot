package bot

import (
	"fmt"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
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

	refs := make([]telegram.ChatRef, len(admins))
	for i, admin := range admins {
		refs[i] = admin.User.Ref()
	}

	return refs, nil
}

func (bot *T) SendFiles(chat telegram.ChatRef, files []dvach.File) {
	for _, file := range files {
		var (
			url       string
			mediaType telegram.MediaType
		)

		switch file.Type {
		case dvach.Webm:
			conv, err := bot.conv.Get(file.URL())
			if err != nil {
				log.Warningf("Webm %s failed to convert: %s", file.URL(), err)
				continue
			}

			url = conv
			mediaType = telegram.Video

		case dvach.Mp4:
			url = file.URL()
			mediaType = telegram.Video

		default:
			url = file.URL()
			mediaType = telegram.Photo
		}

		if err := <-bot.Send(telegram.SendMediaRequest{
			Chat: chat,
			Media: []telegram.InputMedia{
				{
					Type:  mediaType,
					Media: url,
				},
			},
		}, nil); err != nil {
			log.Warningf("Failed to send file %s to %s: %s", url, chat, err)
			if err := <-bot.Send(telegram.SendMessageRequest{
				Chat:      chat,
				ParseMode: telegram.HTML,
				Text:      fmt.Sprintf(`<a href="%s">[A]</a>`, html.Escape(file.URL())),
			}, nil); err != nil {
				log.Errorf("Failed to send link %s to %s: %s", file.URL(), chat, err)
			}
		}
	}
}

func (bot *T) SendPost(chat telegram.ChatRef, post html.Post) error {
	text := html.Chunk(post, chunkSize)
	for _, part := range text {
		err := <-bot.Send(telegram.SendMessageRequest{
			Chat:      chat,
			ParseMode: telegram.HTML,
			Text:      part,
		}, nil)

		if err != nil {
			log.Errorf("Failed to send text to %s: %s", chat, err)
			return err
		}
	}

	bot.SendFiles(chat, post.Files)
	return nil
}
