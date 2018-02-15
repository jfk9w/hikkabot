package main

import (
	"fmt"
	"strings"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

func Controller(bot telegram.BotAPI) util.Handle {
	h := util.NewHandle()
	go func() {
		defer h.Reply()
		log.Debug("CTRL start")
		for {
			select {
			case u := <-bot.In():
				go handleUpdate(bot, u)

			case <-h.C:
				log.Debug("CTRL stop")
				return
			}
		}
	}()

	return h
}

func handleUpdate(bot telegram.BotAPI, u telegram.Update) {
	msg := u.Message
	if msg != nil {
		cmd, params := parseCommand(bot, msg)
		ctx := context{
			bot:    bot,
			source: telegram.ChatRef{ID: msg.Chat.ID},
			userID: msg.From.ID,
			params: params,
		}

		switch cmd {
		case "/subscribe", "/sub":
			ctx.log().Debug("CTRL subscribe")
			subscribe(ctx)

		case "/unsubscribe", "/unsub":
			ctx.log().Debug("CTRL unsubscribe")
			unsubscribe(ctx)

		case "/status":
			ctx.log().Debug("CTRL status")
			status(ctx)
		}
	}
}

type context struct {
	bot    telegram.BotAPI
	source telegram.ChatRef
	userID telegram.UserID
	params []string
}

func (ctx context) log() *log.Entry {
	return log.WithFields(log.Fields{
		"source": ctx.source.Key(),
		"userID": ctx.userID,
		"params": ctx.params,
	})
}

func subscribe(ctx context) {
	var chat telegram.ChatRef
	switch len(ctx.params) {
	case 0:
		ctx.bot.SendMessage(telegram.SendMessageRequest{
			Chat:      ctx.source,
			ParseMode: telegram.Markdown,
			Text:      "#info\nUsage: `/subscribe` THREAD_URL",
		}, true, nil)
		return

	case 1:
		chat = ctx.source

	case 2:
		chat = telegram.ChatRef{Username: ctx.params[1]}
	}

	board, thread, err := dvach.ParseThreadURL(ctx.params[0])
	if err != nil {
		ctx.bot.SendMessage(telegram.SendMessageRequest{
			Chat: ctx.source,
			Text: fmt.Sprintf("#info\nInvalid thread URL: %s", ctx.params[0]),
		}, true, nil)
		return
	}

	if err = service.CheckAccess(ctx.userID, chat); err != nil {
		ctx.bot.SendMessage(telegram.SendMessageRequest{
			Chat: ctx.source,
			Text: "#info\nOperation forbidden. Reason: " + err.Error(),
		}, true, nil)

		return
	}

	service.Subscribe(chat, board, thread)
}

func unsubscribe(ctx context) {
	var chat telegram.ChatRef
	if len(ctx.params) == 1 {
		chat = telegram.ChatRef{
			Username: ctx.params[0],
		}
	} else {
		chat = ctx.source
	}

	if err := service.CheckAccess(ctx.userID, chat); err != nil {
		ctx.bot.SendMessage(telegram.SendMessageRequest{
			Chat: ctx.source,
			Text: "#info\nOperation forbidden. Reason: " + err.Error(),
		}, true, nil)

		return
	}

	service.Unsubscribe(chat)
}

func status(ctx context) {
	ctx.bot.SendMessage(telegram.SendMessageRequest{
		Chat: ctx.source,
		Text: "#info\nWhile you're dying I'll be still alive\nAnd when you're dead I will be still alive\nStill alive\nS T I L L A L I V E",
	}, true, nil)
}

func parseCommand(bot telegram.BotAPI, msg *telegram.Message) (string, []string) {
	for _, entity := range msg.Entities {
		if entity.Type == "bot_command" {
			end := entity.Offset + entity.Length

			cmd := msg.Text[entity.Offset:end]
			cmd = strings.Replace(cmd, "@"+bot.Me().Username, "", 1)

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
