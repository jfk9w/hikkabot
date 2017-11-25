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

	bot := telegram.NewBotAPI(httpClient, cfg.Token)
	client := dvach.NewAPI(httpClient)

	service.Init(bot, client, cfg.DBFilename)

	ctl := NewController()
	ctl.Init(bot, client)
	ctl.Start()

	SignalHandler().Wait()
	ctl.Stop()

	sawmill.Notice("Exit")
}
