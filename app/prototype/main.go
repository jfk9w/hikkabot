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

	handler := subscription.NewHandler(bot, ctx, nil, 20*time.Second, services.All)
	bot.Listen(handler.CommandListener())
}
