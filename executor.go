package main

import (
	"strconv"
	"sync"

	"github.com/jfk9w/tele2ch/telegram"
)

type listener func(message *telegram.Message)

type executor struct {
	bot       *telegram.BotAPI
	subs      *Subs
	listeners map[string]listener
}

func newExecutor(bot *telegram.BotAPI, subs *Subs) *executor {
	return &executor{
		bot:       bot,
		subs:      subs,
		listeners: make(map[string]listener),
	}
}

func (svc *executor) run(chatId telegram.ChatID, cmd string, params []string) {
	switch cmd {
	case "/subscribe":

	}
}

func (svc *executor) subscribe(mgmt telegram.ChatID, params []string) {
	switch len(params) {
	case 0:
		svc.askForThreadLink(mgmt, func(resp *telegram.Response, err error) {
			if resp.Ok {
				message := new(telegram.Message)
				err = resp.Parse(message)
				if err == nil {
					svc.listen(message.ID, func(message *telegram.Message) {
						threadLink := message.Text
						svc.subscribe0(mgmt, telegram.ChatRef{
							ID: mgmt,
						}, threadLink)
					})
				}
			}
		})

	case 1:
		threadLink := params[0]
		svc.subscribe0(mgmt, telegram.ChatRef{
			ID: mgmt,
		}, threadLink)

	case 2:
		threadLink := params[1]
		chat := telegram.ChatRef{
			Username: params[2],
		}

		svc.subscribe0(mgmt, chat, threadLink)
	}
}

func (svc *executor) subscribe0(mgmt telegram.ChatID, chat telegram.ChatRef, threadLink string) {

}

func (svc *executor) askForThreadLink(mgmt telegram.ChatID, handler telegram.ResponseHandler) {
	svc.bot.SendMessage(telegram.SendMessageRequest{
		Chat: telegram.ChatRef{
			ID: mgmt,
		},
		Text: "Введите ссылку на тред",
		ReplyMarkup: telegram.ForceReply{
			ForceReply: true,
			Selective:  true,
		},
	}, handler, true)
}

func (svc *executor) listen(messageId telegram.MessageID, l listener) {
	key := strconv.Itoa(int(messageId))
	svc.listeners[key] = l
}

func (svc *executor) reply(message *telegram.Message) {
	key := strconv.Itoa(int(message.ReplyToMessage.ID))
	if l, ok := svc.listeners[key]; ok {
		delete(svc.listeners, key)
		l(message)
	}
}
