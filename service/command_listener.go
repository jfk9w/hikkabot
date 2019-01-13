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
	services []Service
}

func (scl *SubscribeCommandListener) SetBot(b *telegram.Bot) *SubscribeCommandListener {
	scl.b = b
	return scl
}

func (scl *SubscribeCommandListener) Add(services ...Service) *SubscribeCommandListener {
	if scl.services == nil {
		scl.services = make([]Service, 0)
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
		chatStr string
		options string
	)

	if len(tokens) > 1 {
		chatStr = tokens[1]
	}

	if len(tokens) > 2 {
		options = tokens[2]
	}

	chat, err := scl.elevate(c, chatStr)
	if err != nil {
		c.ErrorReply(err)
		return
	}

	for _, svc := range scl.services {
		err = svc.Subscribe(tokens[0], chat, options)
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

func (scl *SubscribeCommandListener) elevate(c *telegram.Command, chatStr string) (*telegram.Chat, error) {
	var chat *telegram.Chat
	if chatStr == "" || chatStr == "." {
		chat = c.Chat
	} else {
		var chatID telegram.ChatID
		chatID, err := telegram.ParseID(chatStr)
		if err != nil {
			chatID = telegram.Username(chatStr)
		}

		chat, err = scl.b.GetChat(chatID)
		if err != nil {
			return nil, err
		}
	}

	id := chat.ID
	if chat.Type == telegram.PrivateChat {
		if chat.ID == c.User.ID {
			return chat, nil
		}
	} else {
		admins, err := scl.b.GetChatAdministrators(id)
		if err != nil {
			return nil, err
		}

		for _, admin := range admins {
			if admin.User.ID == c.User.ID {
				return chat, nil
			}
		}
	}

	return nil, ErrForbidden
}
