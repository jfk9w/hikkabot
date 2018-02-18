package controller

import (
	"fmt"
	"strings"

	"github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

func Start(db storage.T, bot telegram.BotAPI) util.Handle {
	h := util.NewHandle()
	subs := make(users)
	go func() {
		defer h.Reply()
		log.Debug("CTRL start")
		for {
			select {
			case u := <-bot.In():
				handleUpdate(db, bot, subs, u)

			case <-h.C:
				log.Debug("CTRL stop")
				return
			}
		}
	}()

	return h
}

func handleUpdate(db storage.T, bot telegram.BotAPI, subs users,
	u telegram.Update) {

	msg := u.Message
	if msg != nil {
		cmd, params := parseCommand(bot, msg)
		ctx := &context{
			bot:    bot,
			subs:   subs,
			source: telegram.ChatRef{ID: msg.Chat.ID},
			userID: msg.From.ID,
			params: params,
		}

		switch cmd {
		case "/subscribe", "/sub":
			ctx.log().Debug("CTRL subscribe")
			if subscribe(ctx) {

			}

		case "/unsubscribe", "/unsub":
			ctx.log().Debug("CTRL unsubscribe")
			unsubscribe(ctx)

		case "/status":
			ctx.log().Debug("CTRL status")
			status(ctx)
		}
	}
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
