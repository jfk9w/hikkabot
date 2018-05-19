package frontend

import (
	"strconv"

	"fmt"

	"github.com/jfk9w-go/telegram"
)

type Command struct {
	string
	params []string

	bot        Bot
	chat, user telegram.ChatRef
}

var emptyCommand = Command{}

func (cmd Command) String() string {
	return fmt.Sprintf("%s %s %s %s", cmd.chat, cmd.user, cmd.string, cmd.params)
}

func (cmd Command) arity() int {
	return len(cmd.params)
}

func (cmd Command) param(idx int) string {
	if cmd.arity() > idx {
		return cmd.params[idx]
	}

	return ""
}

func (cmd Command) channelOrSelf(idx int) telegram.ChatRef {
	return telegram.FirstChatRef(cmd.param(idx), cmd.chat)
}

func (cmd Command) reply(text string, args ...interface{}) {
	go cmd.bot.SendText(cmd.chat, text, args...)
}

func (cmd Command) requireArity(n int) bool {
	if cmd.arity() < n {
		cmd.reply("%d arguments required", n)
		return false
	}

	return true
}

func (cmd Command) requireAdmin(chat telegram.ChatRef) ([]telegram.ChatRef, bool) {
	admins, err := cmd.bot.GetAdmins(chat)
	if err != nil {
		cmd.reply("forbidden: %s", err)
		return nil, false
	}

	for _, admin := range admins {
		if admin == cmd.user {
			return admins, true
		}
	}

	cmd.reply("forbidden")
	return nil, false
}

func (cmd Command) int(idx int, def int) int {
	if r, err := strconv.Atoi(cmd.param(idx)); err == nil {
		return r
	}

	return def
}
