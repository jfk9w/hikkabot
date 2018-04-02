package controller

import (
	tg "github.com/jfk9w/hikkabot/telegram"
	"strconv"
)

func subscribe(ctx *context) {
	var (
		chat = ctx.caller.Chat
		url  = ""
	)

	plen := len(ctx.params)
	if len(ctx.params) > 0 {
		url = ctx.params[0]
		if plen > 1 {
			chat = tg.ChatRef{Username: ctx.params[1]}
		}
	}

	ctx.svc.Subscribe(ctx.caller, chat, url)
}

func unsubscribe(ctx *context) {
	chat := ctx.caller.Chat
	if len(ctx.params) == 1 {
		chat = tg.ChatRef{Username: ctx.params[0]}
	}

	ctx.svc.Unsubscribe(ctx.caller, chat)
}

func status(ctx *context) {
	ctx.svc.Status(ctx.caller)
}

func front(ctx *context) {
    if len(ctx.params) > 0 {
        board := ctx.params[0]

        limit := 10
        if len(ctx.params) == 2 {
        	limit, _ = strconv.Atoi(ctx.params[1])
        }

       	ctx.svc.Front(ctx.caller, board, limit)
    }
}
