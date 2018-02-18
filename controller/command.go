package controller

import (
	"fmt"
	"strings"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

func subscribe(ctx *context) bool {
	var chat telegram.ChatRef
	switch len(ctx.params) {
	case 0:
		ctx.bot.SendMessage(telegram.SendMessageRequest{
			Chat:      ctx.source,
			ParseMode: telegram.Markdown,
			Text: `#info
			Usage: ` + "`/subscribe`" + ` THREAD_URL`,
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
			Text: `#info
			Invalid thread URL: ` + ctx.params[0],
		}, true, nil)
		return
	}

	if ctx.checkAccess(chat) {
		service.Subscribe(chat, board, thread)
		return true
	}

	return false
}

func unsubscribe(ctx *context) bool {
	var chat telegram.ChatRef
	if len(ctx.params) == 1 {
		chat = telegram.ChatRef{
			Username: ctx.params[0],
		}
	} else {
		chat = ctx.source
	}

	if ctx.checkAccess(chat) {
		service.Unsubscribe(chat)
		return true
	}

	return false
}

func status(ctx *context) bool {
	ctx.bot.SendMessage(telegram.SendMessageRequest{
		Chat: ctx.source,
		Text: `#info
		While you're dying I'll be still alive
		And when you're dead I will be still alive
		Still alive
		S T I L L A L I V E`,
	}, true, nil)

	return true
}
