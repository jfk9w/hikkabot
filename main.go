package main

import (
	"net/http"
	"os"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/telegram"
	log "github.com/sirupsen/logrus"
)

func main() {
	InitLogging(cfg)

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	var httpClient = new(http.Client)

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

	client := dvach.NewAPI(httpClient)

	service.Init(bot, client, cfg.DBFilename)

	client.Start()
	ctl := Controller(bot)

	SignalHandler().Wait()
	ctl.Ping()
	bot.Stop()
	client.Stop()
}
