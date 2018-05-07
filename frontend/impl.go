package frontend

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
	"github.com/pkg/errors"
)

type T struct {
	bot  Bot
	dvch Dvach
	back Backend
}

func (front *T) run() {
	for update := range front.bot.UpdateChannel() {
		cmd := front.parseCommand(update.Message)
		if cmd == nil {
			continue
		}

		chat := update.Message.Chat.Ref()
		user := update.Message.From.Ref()
		log.Infof("%s %d %s", chat, user, cmd)

		switch cmd.Command {
		case "status":
			front.bot.SendText(chat, "Alive.")

		case "sub", "subscribe":
			if len(cmd.Params) == 0 {
				front.bot.SendText(chat, "Invalid command.")
				continue
			}

			thread, offset, err := front.parseThread(cmd.Params[0])
			if err != nil {
				front.bot.SendText(chat, "Invalid command: %s", err)
				continue
			}

			hash, err := front.getHash(*thread)
			if err != nil {
				front.bot.SendText(chat, "Unable to load thread title: %s", err)
				continue
			}

			ref := chat
			if len(cmd.Params) > 1 {
				channel := cmd.Params[1]
				if misc.IsFirstRune(channel, '@') {
					ref = telegram.NewChannelRef(cmd.Params[1])
				}
			}

			admins, err := front.checkAccess(user, ref)
			if err != nil {
				front.bot.SendText(chat, "Access denied: %s", err)
				continue
			}

			if err := front.back.Subscribe(ref, *thread, hash, offset); err != nil {
				front.bot.SendText(chat, err.Error())
				continue
			}

			front.bot.NotifyAll(admins,
				"#info\nSubscription OK.\nChat: %s\nThread: %s\nOffset: %d",
				ref, (*thread).URL(), offset)

		case "unsub_all", "unsubscribe_all":
			ref := chat
			if len(cmd.Params) > 0 {
				channel := cmd.Params[0]
				if misc.IsFirstRune(channel, '@') {
					ref = telegram.NewChannelRef(channel)
				}
			}

			admins, err := front.checkAccess(user, ref)
			if err != nil {
				front.bot.SendText(chat, "Access denied: %s", err)
				continue
			}

			if err := front.back.UnsubscribeAll(ref); err != nil {
				front.bot.SendText(chat, err.Error())
				continue
			}

			front.bot.NotifyAll(admins, "#info\nSubscriptions cleared.\nChat: %s", ref)

		case "unsub", "unsubscribe":
			if len(cmd.Params) == 0 {
				front.bot.SendText(chat, "Invalid command.")
				continue
			}

			thread, _, err := front.parseThread(cmd.Params[0])
			if err != nil {
				front.bot.SendText(chat, "Invalid command: %s", err)
				continue
			}

			ref := chat
			if len(cmd.Params) > 1 {
				channel := cmd.Params[1]
				if misc.IsFirstRune(channel, '@') {
					ref = telegram.NewChannelRef(cmd.Params[1])
				}
			}

			admins, err := front.checkAccess(user, ref)
			if err != nil {
				front.bot.SendText(chat, "Access denied: %s", err)
				continue
			}

			if err := front.back.Unsubscribe(ref, *thread); err != nil {
				front.bot.SendText(chat, err.Error())
				continue
			}

			front.bot.NotifyAll(admins,
				"#info\nSubscription stopped.\nChat: %s\nThread: %s",
				ref, (*thread).URL())
		}
	}
}

func (front *T) checkAccess(user, chat telegram.ChatRef) ([]telegram.ChatRef, error) {
	admins, err := front.bot.GetAdmins(chat)
	if err != nil {
		return nil, err
	}

	for _, admin := range admins {
		if admin == user {
			return admins, nil
		}
	}

	return nil, errors.New("forbidden")
}

var postHashRegex = regexp.MustCompile(`#?([A-Za-z]+)(\d+)`)

func (front *T) parseThread(value string) (*dvach.ID, int, error) {
	thread, offset, err := dvach.ParseThread(value)
	if err != nil {
		groups := postHashRegex.FindSubmatch([]byte(value))
		if len(groups) == 3 {
			thread := &dvach.ID{
				Board: strings.ToLower(string(groups[1])),
				Num:   string(groups[2]),
			}

			post, err := front.dvch.Post(*thread)
			if err != nil {
				log.Warningf("Unable to load post %s: %s")
				return nil, 0, err
			}

			if post.Parent != "0" {
				offset, _ = strconv.Atoi(thread.Num)
				offset++

				thread.Num = post.Parent
			}

			return thread, offset, nil
		}

		return nil, 0, err
	}

	return thread, offset, nil
}

func (front *T) getHash(thread dvach.ID) (string, error) {
	post, err := front.dvch.Post(thread)
	if err != nil {
		return "", err
	}

	return html.Hash(post.Subject), nil
}

func (front *T) parseCommand(message *telegram.Message) *ParsedCommand {
	if message == nil {
		return nil
	}

	text := message.Text
	if !misc.IsFirstRune(text, '/') {
		return nil
	}

	tokens := strings.Split(text, " ")
	if len(tokens[0]) <= 1 {
		return nil
	}

	cmd := tokens[0][1:]
	bot, err := front.bot.GetMe()
	if err == nil {
		name := bot.Username
		cmd = strings.Replace(cmd, "@"+name, "", 1)
	}

	return &ParsedCommand{strings.ToLower(cmd), tokens[1:]}
}
