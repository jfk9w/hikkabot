package main

import (
	"fmt"
	"strconv"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/telegram"
)

type listener func(message *telegram.Message)

// Executor service for user commands
type Executor struct {
	bot       *telegram.BotAPI
	listeners map[string]listener
}

// NewExecutor creates a new executor service
func NewExecutor(bot *telegram.BotAPI) *Executor {
	return &Executor{
		bot:       bot,
		listeners: make(map[string]listener),
	}
}

// Run user command
func (svc *Executor) Run(userID telegram.UserID, chatID telegram.ChatID, cmd string, params []string) {
	source := telegram.ChatRef{ID: chatID}
	switch cmd {
	case "/subscribe":
	case "/sub":
		svc.subscribe(userID, source, params)

	case "/unsubscribe":
	case "/unsub":
		svc.unsubscribe(userID, source, params)

	case "/status":
		svc.bot.SendMessage(telegram.SendMessageRequest{
			Chat: source,
			Text: "#info\nWhile you're dying I'll be still alive\nAnd when you're dead I will be still alive\nStill alive\nS T I L L A L I V E",
		}, nil, true)
	}
}

func (svc *Executor) subscribe(userID telegram.UserID, source telegram.ChatRef, params []string) {
	switch len(params) {
	case 0:
		svc.ask(source, func(resp *telegram.Response, err error) {
			if resp.Ok {
				message := new(telegram.Message)
				err = resp.Parse(message)
				if err == nil {
					svc.listen(message.ID, func(message *telegram.Message) {
						threadLink := message.Text
						svc.subscribe0(userID, source, nil, threadLink)
					})
				}
			}
		})

	case 1:
		threadLink := params[0]
		svc.subscribe0(userID, source, nil, threadLink)

	case 2:
		threadLink := params[0]
		channel := params[1]
		svc.subscribe0(userID, source, &channel, threadLink)
	}
}

func (svc *Executor) subscribe0(userID telegram.UserID, source telegram.ChatRef, channel *string, threadLink string) {
	board, threadID, err := dvach.ParseThreadURL(threadLink)
	if err != nil {
		svc.bot.SendMessage(telegram.SendMessageRequest{
			Chat: source,
			Text: fmt.Sprintf("#info\nInvalid thread URL: %s", threadLink),
		}, nil, true)

		return
	}

	var chat telegram.ChatRef
	if channel != nil {
		chat = telegram.ChatRef{
			Username: *channel,
		}
	} else {
		chat = source
	}

	if err = service.CheckAccess(userID, chat); err != nil {
		svc.bot.SendMessage(telegram.SendMessageRequest{
			Chat: source,
			Text: "#info\nOperation forbidden. Reason: " + err.Error(),
		}, nil, true)

		return
	}

	service.Subscribe(chat, board, threadID)
}

func (svc *Executor) unsubscribe(userID telegram.UserID, source telegram.ChatRef, params []string) {
	var chat telegram.ChatRef
	if len(params) == 1 {
		chat = telegram.ChatRef{
			Username: params[0],
		}
	} else {
		chat = source
	}

	if err := service.CheckAccess(userID, chat); err != nil {
		svc.bot.SendMessage(telegram.SendMessageRequest{
			Chat: source,
			Text: "#info\nOperation forbidden. Reason: " + err.Error(),
		}, nil, true)

		return
	}

	service.Unsubscribe(chat)
}

func (svc *Executor) ask(source telegram.ChatRef, onReply telegram.ResponseHandler) {
	svc.bot.SendMessage(telegram.SendMessageRequest{
		Chat: source,
		Text: "Введите ссылку на тред.",
		ReplyMarkup: telegram.ForceReply{
			ForceReply: true,
			Selective:  true,
		},
	}, onReply, true)
}

func (svc *Executor) listen(messageID telegram.MessageID, l listener) {
	key := strconv.Itoa(int(messageID))
	svc.listeners[key] = l
}

func (svc *Executor) OnReply(message *telegram.Message) {
	key := strconv.Itoa(int(message.ReplyToMessage.ID))
	if l, ok := svc.listeners[key]; ok {
		delete(svc.listeners, key)
		l(message)
	}
}
