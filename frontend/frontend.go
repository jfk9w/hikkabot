package frontend

import (
	"fmt"
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
	superusers []telegram.ChatID
}

func Init(svc *service.T, superusers []telegram.ChatID) *T {
	var fe = &T{svc, superusers}
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

	case "unsub", "clear":
		svc.uns

	case "front", "search":
		svc.search(command)

	case "status":
		svc.status(command)

	case "exec":
		svc.exec(command)

	case "query":
		svc.query(command)
	}
}

func (svc *T) exec(command telegram.Command) {
	var err = svc.superuser(command.User)
	if !svc.check(command, err) {
		return
	}

	var updated int64
	updated, err = svc.Exec(strings.Join(command.Args, " "))
	if !svc.check(command, err) {
		return
	}

	var text = fmt.Sprintf("updated %d rows", updated)
	svc.SendMessage(command.Chat, text, nil)
}

func (svc *T) query(command telegram.Command) {
	var err = svc.superuser(command.User)
	if !svc.check(command, err) {
		return
	}

	var report [][]string
	report, err = svc.Query(strings.Join(command.Args, " "))
	if !svc.check(command, err) {
		return
	}

	var b = new(strings.Builder)
	for _, row := range report {
		for _, col := range row {
			b.WriteString(col)
			b.WriteRune(',')
		}

		b.WriteRune('\n')
	}

	svc.SendMessage(command.Chat, b.String(), nil)
}

func (svc *T) status(command telegram.Command) {
	svc.SendMessage(command.Chat, "alive", nil)
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
		svc.SendMessage(command.Chat, part, &telegram.MessageOpts{
			SendOpts: &telegram.SendOpts{
				ParseMode:           telegram.HTML,
				DisableNotification: true,
			},
			DisableWebPagePreview: true,
		})
	}
}

func (svc *T) unsubscribe(command telegram.Command) {
	var (
		target telegram.ChatID
		err    error
	)

	var v = command.Arg(0, "")
	if v == "" {
		target = command.Chat
	} else {
		var ref, err = telegram.ParseRef(v)
		if !svc.check(command, err) {
			return
		}

		var chat *telegram.Chat
		chat, err = svc.GetChat(ref)
		if !svc.check(command, err) {
			return
		}

		target = chat.ID
	}

	err = svc.SuspendAccount(target)
	svc.check(command, err)
}

func (svc *T) subscribe(command telegram.Command, mode string) {
	var (
		ref    dvach.Ref
		target telegram.ChatID
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
		var ref, err = telegram.ParseRef(v)
		if !svc.check(command, err) {
			return
		}

		var chat *telegram.Chat
		chat, err = svc.GetChat(ref)
		if !svc.check(command, err) {
			return
		}

		target = chat.ID
	}

	err = svc.authorize(command.User, target)
	if !svc.check(command, err) {
		return
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

func (svc *T) superuser(user telegram.ChatID) error {
	for _, superuser := range svc.superusers {
		if user == superuser {
			return nil
		}
	}

	return errors.New("forbidden")
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
