package main

import (
	"net/http"

	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
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

	state := GetDomains(cfg)
	state.Init(bot, client)

	ctl := NewController(state)
	ctl.Init(bot, client)
	ctl.Start()

	SignalHandler().Wait()
	ctl.Stop()

	sawmill.Notice("Exit")
}
