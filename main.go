package main

import (
	"github.com/jfk9w-go/lego/json"

	aconvert "github.com/jfk9w-go/aconvert-api"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/media"
	"github.com/jfk9w/hikkabot/services"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/subscription"
	"github.com/jfk9w/hikkabot/util"
)

func main() {
	config := new(struct {
		AdminID        telegram.ID
		Aliases        map[telegram.Username]telegram.ID
		Storage        storage.SQLConfig
		UpdateInterval json.Duration
		Telegram       struct{ Token string }
		Media          struct {
			media.Config
			Aconvert aconvert.Config
		}

		Reddit reddit.Config
		Dvach  struct{ Usercode string }
	})

	util.ReadJSON("bin/config_dev.json", config)
	bot := telegram.NewBot(nil, config.Telegram.Token)
	aconvertClient := aconvert.NewClient(nil, &config.Media.Aconvert)
	mediaManager := media.NewManager(config.Media.Config, aconvertClient)
	defer mediaManager.Shutdown()

	ctx := subscription.Context{
		MediaManager: mediaManager,
		DvachClient:  dvach.NewClient(nil, config.Dvach.Usercode),
		RedditClient: reddit.NewClient(nil, &config.Reddit),
	}

	storage := storage.NewSQL(config.Storage)
	handler := subscription.NewHandler(bot, ctx, storage, config.UpdateInterval.Value(), services.All, config.Aliases)
	go bot.Send(config.AdminID, &telegram.Text{Text: "⬆️"}, nil)
	bot.Listen(handler.CommandListener())
}
