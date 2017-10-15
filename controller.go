package main

import (
	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
	"time"
)

type Controller struct {
	bot   *telegram.BotAPI
	dvach *dvach.API
}

func SetUp(cfg Config) *Controller {
	bot := telegram.NewBotAPI(HttpClient, cfg.Token)
	dvachClient := dvach.NewAPI(HttpClient, dvach.APIConfig{
		ThreadFeedTimeout: time.Minute,
	})

	return &Controller{
		bot,
		dvachClient,
	}
}

func (svc *Controller) Start() {
	svc.bot.Start(&telegram.GetUpdatesRequest{
		Timeout:        3,
		AllowedUpdates: []string{"message"},
	})
}

func (svc *Controller) Stop() {
	svc.bot.Stop(false)
}
