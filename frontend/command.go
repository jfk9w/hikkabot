package frontend

import (
	"strconv"

	"fmt"

	"github.com/jfk9w-go/telegram"
)

type command struct {
	string
	params []string

	bot        Bot
	chat, user telegram.ChatRef
}

var emptyCommand = command{}

func (cmd command) String() string {
	return fmt.Sprintf("%s %s %s %s", cmd.chat, cmd.user, cmd.string, cmd.params)
}

func (cmd command) arity() int {
	return len(cmd.params)
}

func (cmd command) param(idx int) string {
	if cmd.arity() > idx {
		return cmd.params[idx]
	}

	return ""
}

func (cmd command) channelOrSelf(idx int) telegram.ChatRef {
	return telegram.FirstChatRef(cmd.param(idx), cmd.chat)
}

func (cmd command) reply(text string, args ...interface{}) {
	go cmd.bot.SendText(cmd.chat, text, args...)
}

func (cmd command) requireArity(n int) bool {
	if cmd.arity() < n {
		cmd.reply("%d arguments required", n)
		return false
	}

	return true
}

func (cmd command) requireAdmin(chat telegram.ChatRef) ([]telegram.ChatRef, bool) {
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

func (cmd command) int(idx int, def int) int {
	if r, err := strconv.Atoi(cmd.param(idx)); err == nil {
		return r
	}

	return def
}
