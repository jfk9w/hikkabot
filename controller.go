package main

import (
	"strings"

	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/util"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/phemmer/sawmill"
)

// Controller encapsulates all business logic
type Controller struct {
	bot      *telegram.BotAPI
	client   *dvach.API
	executor *Executor
	halt     util.Hook
	done     util.Hook
}

// NewController creates an empty controller
func NewController() *Controller {
	return &Controller{
		halt: util.NewHook(),
		done: util.NewHook(),
	}
}

// Init injects necessary dependencies
func (svc *Controller) Init(bot *telegram.BotAPI, client *dvach.API) {
	svc.bot = bot
	svc.client = client
	svc.executor = NewExecutor(bot)
}

var getUpdatesRequest = telegram.GetUpdatesRequest{
	Timeout:        60,
	AllowedUpdates: []string{"message"},
}

// Start the controller
func (svc *Controller) Start() {

	service.Start()

	go func() {
		defer svc.done.Send()

		uc := svc.bot.GetUpdatesChan(getUpdatesRequest)
		for {
			select {
			case u := <-uc:
				msg := u.Message
				if msg != nil {
					chatID := msg.Chat.ID
					userID := msg.From.ID
					cmd, params := svc.parseCommand(msg)
					switch {
					case len(cmd) > 0:
						svc.executor.Run(userID, chatID, cmd, params)

					case msg.ReplyToMessage != nil:
						svc.executor.OnReply(msg)
					}
				}

			case <-svc.halt:
				return
			}
		}
	}()

	sawmill.Notice("controller started")
}

// Stop the controller
func (svc *Controller) Stop() {

	svc.halt.Send()
	svc.done.Wait()

	sawmill.Notice("controller stopped")

	svc.bot.Stop()
	service.Stop()
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
