package service

import (
	"sync"

	"html"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/schedx"
	"github.com/jfk9w-go/gox/syncx"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/telegram"
)

type (
	Scheduler = schedx.T
	Telegram  = telegram.T
	Dvach     = dvach.API
	Aconvert  = aconvert.Balancer
)

type Context struct {
	Telegram
	Dvach
	Aconvert
}

func (ctx *Context) ParentThread(ref dvach.Ref) (dvach.Ref, error) {
	post, err := ctx.Dvach.Post(ref)
	if err != nil {
		return dvach.Ref{}, err
	}

	if post.Parent != 0 {
		ref.Num = post.Parent
		ref.NumString = post.NumString
	}

	return ref, nil
}

func (ctx *Context) NotifyAdministrators(chat *telegram.Chat, text string) {
	var admins = make([]telegram.ChatID, 0)
	if chat.Type == telegram.PrivateChatType {
		admins = append(admins, chat.ID)
	} else {
		members, err := ctx.GetChatAdministrators(chat.ID)
		if err != nil {
			return
		}

		for _, member := range members {
			if !member.User.IsBot {
				admins = append(admins, member.User.ID)
			}
		}
	}

	for _, id := range admins {
		go ctx.SendMessage(id, text, nil)
	}
}

func (ctx *Context) SendPost(chat *telegram.Chat, hashtag string, post *dvach.Post) error {
	var (
		group sync.WaitGroup
		files = syncx.NewMap()
		err   error
	)

	group.Add(len(post.Files))
	for _, dfile := range post.Files {
		go func(dfile *dvach.File) {
			var (
				url  = dfile.URL()
				file = new(httpx.File)
				err  = ctx.Dvach.Get(url, nil, file)
			)

			if dfile.Type == dvach.Webm {
				url, err = ctx.Convert(file)
				if err != nil {
					goto wrap
				}

				err = ctx.Aconvert.Get(url, nil, file)
			}

		wrap:
			if err == nil {
				files.Put(url, file)
			}

			group.Done()
		}(dfile)
	}

	var (
		sendOpts = &telegram.SendOpts{
			ParseMode:           telegram.HTML,
			DisableNotification: true,
		}

		messageOpts = &telegram.MessageOpts{
			SendOpts: sendOpts,
		}
	)

	parts := text.FormatPost(text.Post{post, hashtag})
	for _, part := range parts {
		_, err = ctx.SendMessage(chat.ID, part, messageOpts)
		if err != nil {
			return err
		}
	}

	group.Wait()
	for _, dfile := range post.Files {
		var (
			link     = `<a href="` + html.EscapeString(dfile.URL()) + `">[A]</a>`
			sendOpts = &telegram.SendOpts{
				ParseMode:           telegram.HTML,
				DisableNotification: true,
			}
		)

		if any, ok := files.Get(dfile.URL()); ok {
			var (
				file      = any.(*httpx.File)
				mediaOpts = &telegram.MediaOpts{
					SendOpts: sendOpts,
					Caption:  link,
				}
			)

			switch dfile.Type {
			case dvach.Gif, dvach.Webm, dvach.Mp4:
				_, err = ctx.SendVideo(chat.ID, file, &telegram.VideoOpts{
					MediaOpts: mediaOpts,
					Duration:  *dfile.DurationSecs,
					Width:     *dfile.Width,
					Height:    *dfile.Height,
				})

			default:
				_, err = ctx.SendPhoto(chat.ID, file, mediaOpts)
			}
		} else {
			_, err = ctx.SendMessage(chat.ID, link, messageOpts)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
