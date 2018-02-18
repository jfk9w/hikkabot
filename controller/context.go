package controller

import (
	"sync"

	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/telegram"
)

type context struct {
	mgmt   *system
	source telegram.ChatRef
	userID telegram.UserID
	params []string
}

func (ctx *context) log() *log.Entry {
	return log.WithFields(log.Fields{
		"source": ctx.source.Key(),
		"userID": ctx.userID,
		"params": ctx.params,
	})
}

func (ctx *context) checkAccess(target telegram.ChatRef) bool {
	if !chat.IsChannel() && int64(target.ID) == int64(ctx.userID) {
		return true
	}

	admins, err := ctx.bot.GetChatAdministrators(chat)
	if err == nil {
		for _, admin := range admins {
			if admin.User.ID == ctx.userID &&
				(admin.Status == "creator" ||
					admin.Status == "administrator" && admin.CanPostMessages) {
				return true
			}
		}
	}

	ctx.bot.SendMessage(telegram.SendMessageRequest{
		Chat:      ctx.source,
		ParseMode: telegram.Markdown,
		Text:      "#info\nOperation forbidden.",
	}, true, nil)

	return false
}
