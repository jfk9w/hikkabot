package main

import (
	"net/http"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/phemmer/sawmill"
)

func main() {
	defer func() {
		sawmill.CheckPanic()
		sawmill.Stop()
	}()

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	var httpClient = new(http.Client)

	InitLogging(cfg)

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
	client.Stop()

	sawmill.Notice("exit")
}
