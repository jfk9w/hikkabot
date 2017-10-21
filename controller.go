package main

import (
	"net/http"
	"strings"

	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
)

var httpClient = new(http.Client)

type controller struct {
	bot      *telegram.BotAPI
	dvach    *dvach.API
	executor *executor
	stop     chan struct{}
}

func InitController(cfg *Config) *controller {
	bot := telegram.NewBotAPI(httpClient, cfg.Token)

	var subs *Subs
	if len(cfg.DBFilename) > 0 {
		subs, err := LoadSubs(cfg.DBFilename)
		if err != nil {
			panic(err)
		}
	} else {
		subs := NewSubs()
	}

	ctl := &controller{
		bot:      bot,
		dvach:    dvach.NewAPI(httpClient),
		executor: newExecutor(bot, subs),
		stop:     make(chan struct{}, 1),
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
		uc := svc.bot.GetUpdates(getUpdatesRequest)
		for {
			select {
			case u := <-uc:
				msg := u.Message
				cmd, params := svc.parseCommand(msg)
				switch {
				case len(cmd) > 0:
					svc.executor.run(cmd, params)

				case msg.ReplyToMessage != nil:
					svc.executor.reply(msg)
				}

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

func (svc *controller) parseCommand(msg *telegram.Message) (string, []string) {
	for _, entity := range msg.Entities {
		if entity.Type == "bot_command" {
			end := entity.Offset + entity.Length

			cmd := msg.Text[entity.Offset:end]
			cmd = strings.Replace(cmd, "@"+svc.bot.Me.Username, "", 1)

			params0 := strings.Split(msg.Text[end:], " ")
			params := make([]string, 0)
			for _, param := range params0 {
				if len(param) > 0 {
					params = append(params, param)
				}
			}

			return cmd, params
		}
	}

	return "", nil
}
