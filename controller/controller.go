package controller

import (
	"strings"

	"github.com/jfk9w/hikkabot/service"
	tg "github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

func Start(bot tg.BotAPI, svc *service.T) util.Handle {
	h := util.NewHandle()
	go func() {
		defer func() {
			log.Info("CTRL stop")
			h.Reply()
		}()

		log.Info("CTRL start")
		for {
			select {
			case u := <-bot.In():
				handleUpdate(bot, svc, u)

			case <-h.C:
				return
			}
		}
	}()

	return h
}

type context struct {
	svc    *service.T
	caller service.Caller
	params []string
}

func (ctx *context) log() *log.Entry {
	return log.WithFields(log.Fields{
		"caller": ctx.caller.UserID,
		"params": ctx.params,
	})
}

func handleUpdate(bot tg.BotAPI, svc *service.T, u tg.Update) {
	msg := u.Message
	if msg != nil {
		cmd, params := parseCommand(bot, msg)
		caller := service.Caller{
			Chat:   tg.ChatRef{ID: msg.Chat.ID},
			UserID: msg.From.ID,
		}

		ctx := &context{
			svc:    svc,
			caller: caller,
			params: params,
		}

		switch cmd {
		case "/subscribe", "/sub":
			ctx.log().Info("CTRL subscribe")
			subscribe(ctx)

		case "/unsubscribe", "/unsub":
			ctx.log().Info("CTRL unsubscribe")
			unsubscribe(ctx)

		case "/status":
			ctx.log().Info("CTRL status")
			status(ctx)

		case "/front":
			ctx.log().Info("CTRL front")
			front(ctx)
		}
	}
}

func parseCommand(bot tg.BotAPI, msg *tg.Message) (string, []string) {
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
