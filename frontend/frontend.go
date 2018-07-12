package frontend

import (
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/common"
	"github.com/jfk9w-go/hikkabot/service"
	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	*service.T
}

func Init(svc *service.T) *T {
	var fe = &T{svc}
	go fe.run()
	return fe
}

func (svc *T) run() {
	for command := range svc.CommandChannel() {
		go svc.process(command)
	}
}

func (svc *T) process(command telegram.Command) {
	switch command.Command {
	case "sub", "subscribe", "all":
		var mode, err = svc.mode(command.Arg(2, service.All))
		if !svc.check(command, err) {
			return
		}

		svc.subscribe(command, mode)

	case "media":
		svc.subscribe(command, service.Media)

	case "text":
		svc.subscribe(command, service.Text)

	case "front", "search":
		svc.search(command)
	}
}

func (svc *T) search(command telegram.Command) {
	var board = command.Arg(0, "")
	if board == "" {
		svc.check(command, errors.New("invalid command"))
		return
	}

	var catalog, err = svc.Catalog(board)
	if !svc.check(command, err) {
		return
	}

	var query = command.Arg(1, "")
	var tokens []string = nil
	if query != "" {
		tokens = strings.Split(query, " ")
	}

	var parts = text.Search(catalog.Threads, tokens)
	for _, part := range parts {
		svc.SendMessage(command.Chat, part, nil)
	}
}

func (svc *T) subscribe(command telegram.Command, mode string) {
	var (
		ref    dvach.Ref
		target telegram.Ref
		err    error
	)

	ref, err = parseRef(command.Arg(0, ""))
	if !svc.check(command, err) {
		return
	}

	var v = command.Arg(1, "")
	if v == "" {
		target = command.Chat
	} else {
		target, err = telegram.ParseRef(v)
		if !svc.check(command, err) {
			return
		}
	}

	err = svc.CreateSubscription(target, ref, mode)
	svc.check(command, err)
}

func (svc *T) mode(value string) (string, error) {
	if value != service.All && value != service.Text && value != service.Media {
		return "", errors.Errorf("invalid mode: %s", value)
	}

	return value, nil
}

func (svc *T) check(command telegram.Command, err error) bool {
	if err != nil {
		go svc.SendMessage(command.Chat, err.Error(), nil)
		return false
	}

	return true
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

func parseRef(thread string) (ref dvach.Ref, err error) {
	ref, err = common.ParseRefTag(thread)
	if err == nil {
		return
	}

	ref, err = dvach.ParseUrl(thread)
	return
}
