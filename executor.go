package main

import (
	"fmt"
	"github.com/jfk9w/tele2ch/dvach"
	"github.com/jfk9w/tele2ch/telegram"
	"strconv"
)

type listener func(message *telegram.Message)

type Executor struct {
	bot       *telegram.BotAPI
	domains   *Domains
	listeners map[string]listener
}

func NewExecutor(bot *telegram.BotAPI, domains *Domains) *Executor {
	return &Executor{
		bot: bot,
		domains:   domains,
		listeners: make(map[string]listener),
	}
}

func (svc *Executor) Run(chatId telegram.ChatID, cmd string, params []string) {
	mgmt := telegram.ChatRef{ID: chatId}

	switch cmd {
	case "/subscribe":
		svc.subscribe(mgmt, params)

	case "/unsubscribe":
		svc.unsubscribe(mgmt, params)

	case "/status":
		svc.bot.SendMessage(telegram.SendMessageRequest{
			Chat: mgmt,
			Text: "Я жив.",
		}, nil, true)
	}
}

func (svc *Executor) subscribe(mgmt telegram.ChatRef, params []string) {
	switch len(params) {
	case 0:
		svc.ask(mgmt, func(resp *telegram.Response, err error) {
			if resp.Ok {
				message := new(telegram.Message)
				err = resp.Parse(message)
				if err == nil {
					svc.listen(message.ID, func(message *telegram.Message) {
						threadLink := message.Text
						svc.subscribe0(mgmt, nil, threadLink)
					})
				}
			}
		})

	case 1:
		threadLink := params[0]
		svc.subscribe0(mgmt, nil, threadLink)

	case 2:
		threadLink := params[1]
		channel := &params[2]
		svc.subscribe0(mgmt, channel, threadLink)
	}
}

func (svc *Executor) subscribe0(mgmt telegram.ChatRef, channel *string, threadLink string) {
	board, threadId, err := dvach.ParseThreadURL(threadLink)
	if err != nil {
		svc.bot.SendMessage(telegram.SendMessageRequest{
			Chat: mgmt,
			Text: fmt.Sprintf("Введена некорректная ссылка: %s", threadLink),
		}, nil, true)

		return
	}

	svc.domains.Subscribe(mgmt, channel, board, threadId)
}

func (svc *Executor) unsubscribe(mgmt telegram.ChatRef, params []string) {
	var channel *string
	if len(params) == 1 {
		channel = &params[0]
	}

	svc.domains.Unsubscribe(mgmt, channel)
}

func (svc *Executor) ask(mgmt telegram.ChatRef, onReply telegram.ResponseHandler) {
	svc.bot.SendMessage(telegram.SendMessageRequest{
		Chat: mgmt,
		Text: "Введите ссылку на тред.",
		ReplyMarkup: telegram.ForceReply{
			ForceReply: true,
			Selective:  true,
		},
	}, onReply, true)
}

func (svc *Executor) listen(messageId telegram.MessageID, l listener) {
	key := strconv.Itoa(int(messageId))
	svc.listeners[key] = l
}

func (svc *Executor) OnReply(message *telegram.Message) {
	key := strconv.Itoa(int(message.ReplyToMessage.ID))
	if l, ok := svc.listeners[key]; ok {
		delete(svc.listeners, key)
		l(message)
	}
}
