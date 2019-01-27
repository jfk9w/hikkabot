package engine

import "github.com/jfk9w-go/hikkabot/common/telegram-bot-api"

type Context struct {
	Telegram
	Dvach
	Aconvert
	Red
}

type EnrichedChat struct {
	*telegram.Chat
	Administrators []telegram.ChatID
}

func (ctx *Context) EnrichChat(id telegram.ChatID) (enriched EnrichedChat, err error) {
	var chat *telegram.Chat
	chat, err = ctx.Telegram.GetChat(id)
	if err != nil {
		return
	}

	var admins = make([]telegram.ChatID, 0)
	if chat.Type == telegram.PrivateChatType {
		admins = append(admins, chat.ID)
	} else {
		var members []telegram.ChatMember
		members, err = ctx.Telegram.GetChatAdministrators(chat.ID)
		if err != nil {
			return
		}

		for _, member := range members {
			if !member.User.IsBot {
				admins = append(admins, member.User.ID)
			}
		}
	}

	enriched.Chat = chat
	enriched.Administrators = admins

	return
}

func (ctx *Context) NotifyAdministrators(id telegram.ChatID, template func(*telegram.Chat) string) {
	var (
		chat, _ = ctx.EnrichChat(id)
		text    = template(chat.Chat)
	)

	for _, id := range chat.Administrators {
		go ctx.SendMessage(id, text, nil)
	}
}
