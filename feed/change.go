package feed

import (
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Change struct {
	Offset int64
	Error  error
}

type changeContext struct {
	chatID   telegram.ChatID
	chat     *telegram.Chat
	adminIDs []telegram.ID
}

func (c *changeContext) getChat(channel Channel) (*telegram.Chat, error) {
	if c.chat != nil {
		return c.chat, nil
	}
	chat, err := channel.GetChat(c.chatID)
	if err != nil {
		return nil, errors.Wrap(err, "on getChat")
	}
	c.chat = chat
	return chat, nil
}

func (c *changeContext) getAdminIDs(channel Channel) ([]telegram.ID, error) {
	if c.adminIDs != nil {
		return c.adminIDs, nil
	}
	chat, err := c.getChat(channel)
	if err != nil {
		return nil, err
	}
	var adminIDs []telegram.ID
	if chat.Type == telegram.PrivateChat {
		adminIDs = []telegram.ID{chat.ID}
	} else {
		admins, err := channel.GetChatAdministrators(c.chatID)
		if err != nil {
			return nil, errors.Wrap(err, "get chat administrators")
		}
		adminIDs = make([]telegram.ID, 0)
		for _, admin := range admins {
			if !admin.User.IsBot {
				adminIDs = append(adminIDs, admin.User.ID)
			}
		}
	}
	c.adminIDs = adminIDs
	return adminIDs, nil
}

func (c *changeContext) checkAccess(channel Channel, userID telegram.ID) error {
	adminIDs, err := c.getAdminIDs(channel)
	if err != nil {
		return err
	}
	for _, adminID := range adminIDs {
		if adminID == userID {
			return nil
		}
	}
	return errors.New("forbidden")
}
