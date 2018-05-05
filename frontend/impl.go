package frontend

import (
	"sync"

	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/backend"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
)

type frontend struct {
	bot       Bot
	back      backend.Backend
	waitGroup *sync.WaitGroup
}

func (f *frontend) Close() error {
	f.waitGroup.Wait()
	return nil
}

func (f *frontend) run() {
	f.waitGroup.Add(1)
	for update := range f.bot.UpdateChannel() {
		cmd := f.ParseCommand(update.Message)
		if cmd == nil {
			continue
		}

		chat := update.Message.Chat.Ref()
		user := update.Message.From.Ref()
		log.Infof("%s %d %s", chat, user, cmd)

		switch cmd.Command {
		case "status":
			f.bot.SendText(chat, "Alive.")

		case "sub", "subscribe":
			if len(cmd.Params) == 0 {
				f.bot.SendText(chat, "Invalid command.")
				continue
			}

			url := cmd.Params[0]
			thread, err := dvach.ParseThread(url)
			if err != nil {
				f.bot.SendText(chat, "Invalid command: %s", err)
				continue
			}

			ref := chat
			if len(cmd.Params) > 1 {
				channel := cmd.Params[1]
				if misc.IsFirstRune(channel, '@') {
					ref = telegram.NewChannelRef(cmd.Params[1])
				}
			}

			admins, err := f.bot.GetAdmins(ref, update.Message.From.Ref())
			if err != nil {
				f.bot.SendText(chat, "Access denied: %s", err)
				continue
			}

			f.back.Subscribe(ref, admins, *thread, 0)

		case "unsub", "unsubscribe":
			ref := chat
			if len(cmd.Params) > 0 {
				channel := cmd.Params[0]
				if misc.IsFirstRune(channel, '@') {
					ref = telegram.NewChannelRef(channel)
				}
			}

			admins, err := f.bot.GetAdmins(ref, update.Message.From.Ref())
			if err != nil {
				f.bot.SendText(chat, "Access denied: %s", err)
				continue
			}

			f.back.UnsubscribeAll(ref, admins)
		}
	}

	f.waitGroup.Done()
}

func (f *frontend) ParseCommand(message *telegram.Message) *ParsedCommand {
	text := message.Text
	if !misc.IsFirstRune(text, '/') {
		return nil
	}

	tokens := strings.Split(text, " ")
	if len(tokens[0]) <= 1 {
		return nil
	}

	cmd := tokens[0][1:]
	bot, err := f.bot.GetMe()
	if err == nil {
		name := bot.Username
		cmd = strings.Replace(cmd, "@"+name, "", 1)
	}

	return &ParsedCommand{strings.ToLower(cmd), tokens[1:]}
}
