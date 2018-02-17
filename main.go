package main

import (
	"net/http"
	"os"

	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/webm"
	log "github.com/sirupsen/logrus"
)

func main() {
	InitLogging(cfg)

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	db := storage.

	httpc := new(http.Client)
	dvach := dv.New(httpc)
	bot, err := telegram.NewBotAPIWithClient(
		httpClient,
		cfg.Token,
		telegram.GetUpdatesRequest{
			Timeout:        60,
			AllowedUpdates: []string{"message"},
		},
	)

	if err != nil {
		panic(err)
	}

	service.Init(bot, dvach, cfg.DBFilename)

	conv, hConv = webm.Converter(webm.Wrap(httpc))
	hCtl := Controller(bot)

	SignalHandler().Wait()
	hCtl.Ping()
	bot.Stop()
	client.Stop()
}
