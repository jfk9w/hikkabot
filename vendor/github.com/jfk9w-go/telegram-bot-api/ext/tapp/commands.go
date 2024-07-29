package tapp

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jfk9w-go/flu/colf"
	"github.com/jfk9w-go/flu/me3x"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/pkg/errors"
)

type Scoped interface {
	CommandScope() CommandScope
}

type CommandScope struct {
	all              bool
	ChatIDs, UserIDs colf.Set[telegram.ID]
}

var Public = CommandScope{all: true}

func (s CommandScope) Wrap(listener telegram.CommandListener) telegram.CommandListenerFunc {
	return func(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
		if s.allow(cmd.Chat.ID, cmd.User.ID) {
			return listener.OnCommand(ctx, client, cmd)
		}

		return nil
	}
}

func (s CommandScope) String() string {
	if s.all {
		return "public"
	}

	var b strings.Builder
	if s.ChatIDs != nil {
		b.WriteString("& chat IDs [" + joinIDs(s.ChatIDs, ", ") + "]")
	}

	if s.UserIDs != nil {
		b.WriteString("& user IDs [" + joinIDs(s.UserIDs, ", ") + "]")
	}

	if b.Len() == 0 {
		return "nobody"
	}

	return strings.Trim(b.String(), "& ")
}

func joinIDs(ids colf.Set[telegram.ID], sep string) string {
	var b strings.Builder
	i := 0
	for id := range ids {
		if i > 0 {
			b.WriteString(sep)
		}

		b.WriteString(id.String())

		i++
	}

	return b.String()
}

func (s CommandScope) allow(chatID, userID telegram.ID) bool {
	if s.all {
		return true
	}

	if s.ChatIDs != nil {
		return s.ChatIDs[chatID]
	}

	if s.UserIDs != nil {
		return s.UserIDs[userID]
	}

	return false
}

func (s CommandScope) Transform(body func(scope telegram.BotCommandScope)) {
	if s.all {
		body(telegram.BotCommandScope{Type: telegram.BotCommandScopeDefault})
		return
	}

	chatIDs := colf.Set[telegram.ID]{}
	colf.AddAll[telegram.ID](&chatIDs, s.UserIDs)
	colf.AddAll[telegram.ID](&chatIDs, s.ChatIDs)

	for chatID := range chatIDs {
		body(telegram.BotCommandScope{Type: telegram.BotCommandScopeChat, ChatID: chatID})
	}
}

func slice(ids map[telegram.ID]bool) []telegram.ID {
	slice := make([]telegram.ID, len(ids))
	i := 0
	for id := range ids {
		slice[i] = id
		i++
	}

	return slice
}

func (s CommandScope) Labels() me3x.Labels {
	labels := me3x.Labels{}
	if s.all {
		return labels.Add("public", true)
	}

	if len(s.ChatIDs) > 0 {
		labels = labels.Add("chatIDs", slice(s.ChatIDs))
	}

	if len(s.UserIDs) > 0 {
		labels = labels.Add("userIDs", slice(s.UserIDs))
	}

	return labels
}

type Commands map[telegram.BotCommandScope]map[string]string

func AddDefaultStart(commands Commands, registry telegram.CommandRegistry, version string) {
	command := "/start"
	description := "Get debug info"

	registry.AddFunc(command, func(ctx context.Context, tgclient telegram.Client, cmd *telegram.Command) error {
		text := fmt.Sprintf("User ID: %d\nChat ID: %s\nBot: %s\nVersion: %s",
			cmd.User.ID, cmd.Chat.ID, tgclient.Username(), version)
		return cmd.Reply(ctx, tgclient, text)
	})

	for _, sc := range commands {
		add(sc, command, description)
	}

	scope := telegram.BotCommandScope{Type: telegram.BotCommandScopeDefault}
	if _, ok := commands[scope]; !ok {
		sc := make(map[string]string)
		add(sc, command, description)
		commands[scope] = sc
	}
}

func (c Commands) AddAll(scope telegram.BotCommandScope, commands ...string) {
	all := make(map[string]string, len(commands))
	for _, command := range commands {
		all[command] = humanizeKey(command)
	}

	c.add(scope, all)
}

func (c Commands) Add(scope telegram.BotCommandScope, command, description string) {
	c.add(scope, map[string]string{command: description})
}

func (c Commands) add(scope telegram.BotCommandScope, commands map[string]string) {
	sc, ok := c[scope]
	if !ok {
		sc = make(map[string]string)
		c[scope] = sc
	}

	for command, description := range commands {
		add(sc, command, description)
	}
}

func add(sc map[string]string, command, description string) {
	if strings.HasPrefix(command, "/") {
		command := command[1:]
		if _, ok := sc[command]; ok {
			log.Panicf("duplicate command handler for %s", command)
		}

		sc[command] = description
	}
}

func (c Commands) Set(ctx context.Context, client telegram.Client) error {
	for scope, commands := range c {
		scope := scope
		botCommands := make([]telegram.BotCommand, 0, len(commands))
		for command, description := range commands {
			botCommands = append(botCommands, telegram.BotCommand{
				Command:     command,
				Description: description,
			})
		}

		if err := client.DeleteMyCommands(ctx, &scope); err != nil {
			return errors.Wrap(err, "delete commands")
		}

		if err := client.SetMyCommands(ctx, &scope, botCommands); err != nil {
			return errors.Wrap(err, "set commands")
		}
	}

	return nil
}
