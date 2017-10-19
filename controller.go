package main

import (
	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
	"net/http"
	"strings"
)

var httpClient = new(http.Client)

type controller struct {
	bot   *telegram.BotAPI
	dvach *dvach.API
	stop  chan struct{}
}

func InitController(cfg *Config) *controller {
	ctl := &controller{
		bot:   telegram.NewBotAPI(httpClient, cfg.Token),
		dvach: dvach.NewAPI(httpClient),
		stop:  make(chan struct{}, 1),
	}

	ctl.start()
	return ctl
}

var getUpdatesRequest = telegram.GetUpdatesRequest{
	Timeout:        60,
	AllowedUpdates: []string{"message"},
}

func (svc *controller) start() {
	svc.bot.Start()
	go func() {
		for {
			select {
			case u := <-svc.bot.GetUpdates(getUpdatesRequest):
				tokens := svc.parseCommand(u.Message)
				response := ""
				for _, t := range tokens {
					response += t + ", "
				}

				svc.bot.SendMessage(telegram.SendMessageRequest{
					Chat: telegram.ChatRef{
						ID: u.Message.Chat.ID,

					},
					Text: response,
				}, nil, true)

			case <-svc.stop:
				svc.stop <- unit
				return
			}
		}
	}()
}

func (svc *controller) Stop() <-chan struct{} {
	<-svc.bot.Stop(false)
	svc.stop <- unit
	return svc.stop
}

func (svc *controller) parseCommand(msg *telegram.Message) []string {
	for _, entity := range msg.Entities {
		if entity.Type == "bot_command" {
			end := entity.Offset + entity.Length

			cmd := msg.Text[entity.Offset:end]
			cmd = strings.Replace(cmd, "@"+svc.bot.Me.Username, "", 1)

			params := strings.Split(msg.Text[end:], " ")

			tokens := make([]string, 1)
			tokens[0] = cmd
			for _, param := range params {
				if len(param) > 0 {
					tokens = append(tokens, param)
				}
			}

			return tokens
		}
	}

	return nil
}
