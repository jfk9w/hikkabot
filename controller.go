package main

import (
	"strings"

	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
	"github.com/phemmer/sawmill"
)

type Controller struct {
	bot      *telegram.BotAPI
	client   *dvach.API
	domains  *Domains
	executor *Executor
	stop     chan struct{}
	done     chan struct{}
}

func NewController(domains *Domains) *Controller {
	return &Controller{
		domains: domains,
	}
}

func (svc *Controller) Init(bot *telegram.BotAPI, client *dvach.API) {
	svc.bot = bot
	svc.client = client
	svc.domains.Init(bot, client)
	svc.executor = NewExecutor(bot, svc.domains)
}

var getUpdatesRequest = telegram.GetUpdatesRequest{
	Timeout:        60,
	AllowedUpdates: []string{"message"},
}

func (svc *Controller) Start() {
	svc.bot.Start()
	svc.domains.RunAll()

	svc.stop = make(chan struct{}, 1)
	svc.done = make(chan struct{}, 1)
	go func() {
		defer func() {
			sawmill.Info("Controller.Stop")
			svc.done <- unit
		}()

		uc := svc.bot.GetUpdatesChan(getUpdatesRequest)
		for {
			select {
			case u := <-uc:
				msg := u.Message
				if msg != nil {
					chatId := msg.Chat.ID
					cmd, params := svc.parseCommand(msg)
					switch {
					case len(cmd) > 0:
						svc.executor.Run(chatId, cmd, params)

					case msg.ReplyToMessage != nil:
						svc.executor.OnReply(msg)
					}
				}

			case <-svc.stop:
				return
			}
		}
	}()

	sawmill.Info("Controller.Start")
}

func (svc *Controller) Stop() {
	svc.stop <- unit
	<-svc.done

	svc.bot.Stop(false)
	svc.domains.Stop()
}

func (svc *Controller) parseCommand(msg *telegram.Message) (string, []string) {
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
