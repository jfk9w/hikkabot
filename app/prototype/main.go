package main

import (
	"time"

	aconvert "github.com/jfk9w-go/aconvert-api"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/app/media"
	"github.com/jfk9w/hikkabot/app/services"
	"github.com/jfk9w/hikkabot/app/subscription"
	"github.com/jfk9w/hikkabot/util"
)

func main() {
	config := new(struct {
		Dvach struct {
			Usercode string `json:"usercode"`
		} `json:"dvach"`
		Aconvert aconvert.Config `json:"aconvert"`
		Telegram struct {
			Token string `json:"token"`
		}
		Reddit reddit.Config `json:"reddit"`
	})

	util.ReadJSON("bin/config_dev.json", config)
	bot := telegram.NewBot(nil, config.Telegram.Token)
	aconvertClient := aconvert.NewClient(nil, &config.Aconvert)
	mediaManager := media.NewManager(media.Config{
		Workers: 4,
		TempDir: "/tmp/hikkabot",
	}, aconvertClient)
	defer mediaManager.Shutdown()

	ctx := subscription.Context{
		MediaManager: mediaManager,
		DvachClient:  dvach.NewClient(nil, config.Dvach.Usercode),
		RedditClient: reddit.NewClient(nil, &config.Reddit),
	}

	sender := subscription.NewSender(bot, telegram.ID(50613409))

	var s subscription.Interface
	cmd := "https://2ch.hk/b/res/200262315.html"
	opts := "meme"
	for _, service := range services.All {
		s0 := service()
		_, err := s0.Parse(ctx, cmd, opts)
		if err == nil {
			s = s0
			break
		}
	}

	if s == nil {
		panic(nil)
	}

	var offset subscription.Offset
	for {
		uc := subscription.NewUpdateCollection(10)
		go s.Update(ctx, offset, uc)
		for u := range uc.C {
			util.Check(sender.Send(u))
			offset = u.Offset
		}

		time.Sleep(time.Minute)
	}
}
