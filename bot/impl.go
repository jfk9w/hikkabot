package bot

import (
	"fmt"

	"io"
	"time"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

// Augmented telegram.Bot
type bot struct {
	io.Closer
	Bot
	aconvert.Cache
}

func NewAugmentedBot(token string, frontend io.Closer) AugmentedBot {
	tgbot := telegram.New(httpx.DefaultClient, telegram.DefaultConfig.WithToken(token), frontend)
	conv := aconvert.WithCache(72*time.Hour, 60*time.Second, 12*time.Hour)
	return &bot{misc.BroadcastCloser(tgbot, conv), tgbot, conv}
}

func (b *bot) SendText(chat telegram.ChatRef, text string, args ...interface{}) {
	text = fmt.Sprintf(text, args...)
	if err := <-b.Send(telegram.SendMessageRequest{
		Chat: chat,
		Text: text,
	}, nil); err != nil {
		log.Warningf("Unable to send message to %s: %s", chat.String(), err)
	}
}

func (b *bot) NotifyAll(chats []telegram.ChatRef, text string, args ...interface{}) {
	text = fmt.Sprintf(text, args...)
	for _, chat := range chats {
		b.SendText(chat, text)
	}
}

func (b *bot) GetAdmins(chat telegram.ChatRef, user telegram.ChatRef) ([]telegram.ChatRef, error) {
	if chat == user {
		return []telegram.ChatRef{user}, nil
	}

	admins, err := b.GetChatAdministrators(chat)
	if err != nil {
		log.Warningf("Cannot get chat %s administrator list: %s", chat.String(), err)
		return nil, errors.New("unable to get chat admin list")
	}

	for _, admin := range admins {
		if admin.User.Ref() == user {
			refs := make([]telegram.ChatRef, len(admins))
			for i, a := range admins {
				refs[i] = a.User.Ref()
			}

			return refs, nil
		}
	}

	return nil, errors.New("forbidden")
}

func (b *bot) SendFiles(chat telegram.ChatRef, files []dvach.File) {
	for _, file := range files {
		var (
			url       string
			mediaType telegram.MediaType
		)

		switch file.Type {
		case dvach.Webm:
			conv, err := b.Get(file.URL())
			if err != nil {
				log.Warningf("Webm %s failed to convert: %s. Skipping", file.URL(), err)
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

		err := <-b.Send(telegram.SendMediaRequest{
			Chat: chat,
			Media: []telegram.InputMedia{
				{
					Type:  mediaType,
					Media: url,
				},
			},
		}, nil)

		if err != nil {
			log.Warningf("Failed to send file %s to %s: %s", url, chat, err)
		}
	}
}

func (b *bot) SendPost(chat telegram.ChatRef, post dvach.Post) error {
	text := html.Chunk(post, chunkSize)
	log.WithFields(logrus.Fields{
		"Post":   post,
		"Chunks": text,
	}).Debugf("Sending post to %s", chat)

	for _, part := range text {
		err := <-b.Send(telegram.SendMessageRequest{
			Chat:      chat,
			ParseMode: telegram.HTML,
			Text:      part,
		}, nil)

		if err != nil {
			log.Errorf("Failed to send text to %s: %s", chat, err)
			return err
		}
	}

	b.SendFiles(chat, post.Files)
	return nil
}
