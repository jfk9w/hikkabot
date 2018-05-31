package bot

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type T struct {
	Bot
	http    httpx.Client
	conv    aconvert.Client
	maxwait time.Duration
}

func Wrap(bot Bot, conv aconvert.Client, config Config) *T {
	return &T{bot, httpx.Configure(config.HttpConfig), conv, time.Duration(config.MaxWait) * time.Millisecond}
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
	var (
		url       = file.URL()
		mediaType telegram.MediaType
		data      io.ReadCloser
		err       error
	)

	switch file.Type {
	case dvach.Webm:
		var resp *aconvert.Response
		resp, err = bot.conv.Convert(aconvert.URL(url))
		if err != nil {
			log.Warningf("Failed to convert video %s, response: %v", url, resp.State)
			return bot.SendLink(chat, file.URL())
		}

		var mp4 string
		mp4, err = resp.URL()
		if err != nil {
			log.Warningf("Failed to convert video %s, response: %v", url, resp.State)
			return bot.SendLink(chat, file.URL())
		}

		url = mp4
		fallthrough

	case dvach.Mp4:
		mediaType = telegram.Video

	default:
		mediaType = telegram.Photo
	}

	data, err = bot.http.Download(url)
	if err != nil {
		log.Warningf("Failed to open download file %s: %s", url, err)
		return bot.SendLink(chat, file.URL())
	}

	name := filepath.Base(file.Path)

	media := &telegram.InputMedia{
		Type:       mediaType,
		Media:      "attach://" + name,
		Duration:   file.DurationSecs,
		Width:      file.Width,
		Height:     file.Height,
		ReadCloser: data,
	}

	if err := <-bot.Send(telegram.SendMediaRequest{
		Chat:                chat,
		Media:               []*telegram.InputMedia{media},
		DisableNotification: true,
	}, nil); err != nil {
		log.Warningf("Failed to send %s as media to %s: %s", url, chat, err)
		return bot.SendLink(chat, file.URL())
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

	wait := sync.WaitGroup{}
	for _, file := range post.Files {
		if file.Type == dvach.Webm {
			wait.Add(1)
			go func(url string) {
				data, err := bot.http.Download(url)
				if err != nil {
					log.Warningf("Failed to open download file %s: %s", data, err)
				}

				bot.conv.Convert(aconvert.ReadCloser{data, file.URL()})
				wait.Done()
			}(file.URL())
		}
	}

	wait.Wait()

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
