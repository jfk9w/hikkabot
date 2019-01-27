package telegram

import (
	"strings"

	"github.com/jfk9w-go/hikkabot/common/gox/mathx"

	"github.com/jfk9w-go/hikkabot/common/gox/utf8x"
)

type Command struct {
	Command    string
	Args       []string
	User, Chat ChatID
}

func (cmd Command) Arg(idx int, def string) string {
	if idx >= len(cmd.Args) {
		return def
	}

	return cmd.Args[idx]
}

func (cmd Command) ArgRange(start, end int) []string {
	if len(cmd.Args) == 0 {
		return nil
	}

	start = mathx.MaxInt(mathx.MinInt(start, len(cmd.Args)-1), 0)
	end = mathx.MaxInt(mathx.MinInt(end, len(cmd.Args)-1), 0)
	return cmd.Args[start:end]
}

func (svc *T) CommandChannel() chan Command {
	var commands = make(chan Command)
	go svc.commandChannel(commands)
	return commands
}

func (svc *T) commandChannel(commands chan Command) {
	var me, err = svc.GetMe()
	if err != nil {
		panic(err)
	}

	for update := range svc.Updates {
		message := coalesce(update.Message, update.EditedMessage)
		if message == nil {
			continue
		}

		if !utf8x.IsFirst(message.Text, '/') {
			continue
		}

		var (
			tokens = strings.Split(message.Text, " ")
			cmd    = tokens[0][1:]
			args   = tokens[1:]
		)

		if cmd == "" {
			continue
		}

		cmd = strings.ToLower(cmd)
		cmd = strings.Replace(cmd, me.Username.Value(), "", 1)

		commands <- Command{cmd, args, message.From.ID, message.Chat.ID}
	}

	close(commands)
}

func coalesce(messages ...*Message) *Message {
	for _, message := range messages {
		if message != nil {
			return message
		}
	}

	return nil
}
