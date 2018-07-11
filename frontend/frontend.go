package frontend

import (
	"github.com/jfk9w-go/hikkabot/service"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	*service.Context
}

func (svc *T) run() {
	for command := range svc.CommandChannel() {
		go svc.process(command)
	}
}

func (svc *T) process(command telegram.Command) {
	switch command.Command {
	case "sub", "subscribe", "all":

	case "media":
	case "fast":

	case "unsub", "unsubscribe":
	case "unsubscribe_all", "clear":
	case "resume":
	}
}

func (svc *T) authorize(user, chat telegram.ChatID) error {
	var admins, err = svc.GetChatAdministrators(chat)
	if err != nil {
		return err
	}

	for _, admin := range admins {
		if admin == user {
			return nil
		}
	}

	return errors.New("forbidden")
}
