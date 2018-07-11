package service

import (
	"html"
	"sync"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/schedx"
	"github.com/jfk9w-go/gox/syncx"
	"github.com/jfk9w-go/hikkabot/common"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/httpx"
	"github.com/jfk9w-go/telegram"
)

type (
	Scheduler = *schedx.T
	Telegram  = *telegram.T
	Dvach     = *dvach.API
	Aconvert  = aconvert.Balancer
)

type Context struct {
	Telegram
	Dvach
	Aconvert
}

type ThreadInfo struct {
	dvach.Ref
	Header string
}

func (ctx *Context) ThreadInfo(ref dvach.Ref) (*ThreadInfo, error) {
	var post, err = ctx.Dvach.Post(ref)
	if err != nil {
		return nil, err
	}

	var header string
	if post.Parent != 0 {
		ref.Num = post.Parent
		ref.NumString = post.NumString

		var thread, err = ctx.Dvach.Thread(ref)
		if err != nil {
			return nil, err
		}

		header = common.Header(&thread.Item)
	} else {
		header = common.Header(&post.Item)
	}

	return &ThreadInfo{ref, header}, nil
}

func (ctx *Context) GetChatAdministrators(id telegram.ChatID) ([]telegram.ChatID, error) {
	var chat, err = ctx.Telegram.GetChat(id)
	if err != nil {
		return nil, err
	}

	var admins = make([]telegram.ChatID, 0)
	if chat.Type == telegram.PrivateChatType {
		admins = append(admins, chat.ID)
	} else {
		var members, err = ctx.Telegram.GetChatAdministrators(chat.ID)
		if err != nil {
			return nil, err
		}

		for _, member := range members {
			if !member.User.IsBot {
				admins = append(admins, member.User.ID)
			}
		}
	}

	return admins, nil
}

func (ctx *Context) NotifyAdministrators(id telegram.ChatID, text string) {
	var admins, _ = ctx.GetChatAdministrators(id)
	for _, id := range admins {
		go ctx.SendMessage(id, text, nil)
	}
}

func (ctx *Context) SendPost(chat *telegram.Chat, header string, post *dvach.Post, feedType FeedType) error {
	var (
		group sync.WaitGroup
		files = syncx.NewMap()
		err   error
	)

	if feedType != Fast {
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
					files.Put(dfile.URL(), file)
				}

				group.Done()
			}(dfile)
		}
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

	if feedType != Media {
		var parts = text.FormatPost(text.Post{post, header})
		for _, part := range parts {
			_, err = ctx.SendMessage(chat.ID, part, messageOpts)
			if err != nil {
				return err
			}
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
				_, err = ctx.SendVideo(chat.ID, file, &telegram.VideoOpts{MediaOpts: mediaOpts})

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
