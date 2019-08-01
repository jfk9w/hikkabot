package subscription

import (
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type auth struct {
	chatID   telegram.ChatID
	userID   telegram.ID
	chat     *telegram.Chat
	adminIDs []telegram.ID
}

func (a *auth) getChat(bot *telegram.Bot) (*telegram.Chat, error) {
	if a.chat != nil {
		return a.chat, nil
	}

	chat, err := bot.GetChat(a.chatID)
	if err != nil {
		return nil, errors.Wrap(err, "on getChat")
	}

	a.chat = chat
	return chat, nil
}

func (a *auth) getAdminIDs(bot *telegram.Bot) ([]telegram.ID, error) {
	if a.adminIDs != nil {
		return a.adminIDs, nil
	}

	chat, err := a.getChat(bot)
	if err != nil {
		return nil, err
	}

	var adminIDs []telegram.ID
	if chat.Type == telegram.PrivateChat {
		adminIDs = []telegram.ID{chat.ID}
	} else {
		admins, err := bot.GetChatAdministrators(a.chatID)
		if err != nil {
			return nil, errors.Wrap(err, "on getChatAdministrators")
		}

		adminIDs = make([]telegram.ID, 0)
		for _, admin := range admins {
			if !admin.User.IsBot {
				adminIDs = append(adminIDs, admin.User.ID)
			}
		}
	}

	a.adminIDs = adminIDs
	return adminIDs, nil
}

func (a *auth) check(bot *telegram.Bot) error {
	if a.userID == telegram.ID(0) {
		return nil
	}

	adminIDs, err := a.getAdminIDs(bot)
	if err != nil {
		return err
	}

	for _, adminID := range adminIDs {
		if adminID == a.userID {
			return nil
		}
	}

	return err
}

func (a *auth) fill(bot *telegram.Bot, c *telegram.Command, chatID telegram.ChatID) {
	if chatID == telegram.Username("") || chatID == telegram.Username(".") || chatID == c.Chat.ID {
		a.chatID = c.Chat.ID
		a.chat = c.Chat
	}

	a.chatID = chatID
}
