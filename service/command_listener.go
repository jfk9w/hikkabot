package service

import (
	"errors"
	"strings"

	telegram "github.com/jfk9w-go/telegram-bot-api"
)

var (
	ErrInvalidFormat = errors.New("invalid format")
	ErrForbidden     = errors.New("forbidden")
)

type SubscribeCommandListener struct {
	b        *telegram.Bot
	services []SubscribeService
}

func (scl *SubscribeCommandListener) Add(services ...SubscribeService) *SubscribeCommandListener {
	if scl.services == nil {
		scl.services = make([]SubscribeService, 0)
	}

	scl.services = append(scl.services, services...)
	return scl
}

func (scl *SubscribeCommandListener) OnCommand(c *telegram.Command) {
	if c.Payload == "" {
		c.ErrorReply(ErrInvalidFormat)
		return
	}

	var (
		tokens  = strings.Split(c.Payload, " ")
		input   = tokens[0]
		chatStr = ""
		chatID  telegram.ID
		options = ""
		err     error
	)

	if len(tokens) > 1 {
		chatStr = tokens[1]
	}

	if len(tokens) > 2 {
		options = tokens[2]
	}

	chatID, err = scl.elevate(c, chatStr)
	if err != nil {
		c.ErrorReply(err)
		return
	}

	for _, svc := range scl.services {
		err = svc.Subscribe(input, chatID, options)
		switch err {
		case nil:
			return

		case ErrInvalidFormat:
			continue

		default:
			c.ErrorReply(err)
			return
		}
	}

	c.ErrorReply(ErrInvalidFormat)
}

func (scl *SubscribeCommandListener) elevate(c *telegram.Command, chatStr string) (telegram.ID, error) {
	var chat *telegram.Chat
	if chatStr == "" || chatStr == "." {
		chat = c.Chat
	} else {
		var chatID telegram.ChatID
		chatID, err := telegram.ParseID(chatStr)
		if err != nil {
			chatID, err = telegram.ParseUsername(chatStr)
			if err != nil {
				return 0, err
			}
		}

		chat, err = scl.b.GetChat(chatID)
		if err != nil {
			return 0, err
		}
	}

	id := chat.ID
	if chat.Type == telegram.PrivateChat {
		if chat.ID == c.User.ID {
			return id, nil
		}
	} else {
		admins, err := scl.b.GetChatAdministrators(id)
		if err != nil {
			return 0, err
		}

		for _, admin := range admins {
			if admin.User.ID == c.User.ID {
				return id, nil
			}
		}
	}

	return 0, ErrForbidden
}
