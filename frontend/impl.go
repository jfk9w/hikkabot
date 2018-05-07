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

	chat, user telegram.ChatRef
}

func (front *T) run() {
	for update := range front.bot.UpdateChannel() {
		cmd := front.parseCommand(update.Message)
		if cmd == nil {
			continue
		}

		front.chat = update.Message.Chat.Ref()
		front.user = update.Message.From.Ref()
		log.Infof("%s %d %s", front.chat, front.user, cmd)

		switch cmd.Command {
		case "status":
			front.status()

		case "sub", "subscribe":
			front.subscribe(cmd.Params)

		case "unsub", "unsubscribe":
			front.unsubscribe(cmd.Params)

		case "unsub_all", "unsubscribe_all":
			front.unsubscribeAll(cmd.Params)

		case "dump":
			front.dump(cmd.Params)
		}
	}
}

func (front *T) dump(params []string) {
	ref := front.chat
	if len(params) > 0 {
		channel := params[0]
		if misc.IsFirstRune(channel, '@') {
			ref = telegram.NewChannelRef(channel)
		}
	}

	admins, err := front.checkAccess(front.user, ref)
	if err != nil {
		front.bot.SendText(front.chat, "Access denied: %s", err)
		return
	}

	entries := front.back.Dump(ref)
	sb := &strings.Builder{}
	sb.WriteString(ref.String())
	sb.WriteString(" entries:\n")
	for thread, entry := range entries {
		if entry.Offset > 0 {
			sb.WriteString(html.Num(thread.Board, strconv.Itoa(entry.Offset)))
		} else {
			sb.WriteString(html.Num(thread.Board, thread.Num))
		}

		sb.WriteRune(' ')
		sb.WriteString(entry.Hash)
		sb.WriteRune('\n')
	}

	front.bot.NotifyAll(admins, sb.String())
}

func (front *T) unsubscribeAll(params []string) {
	ref := front.chat
	if len(params) > 0 {
		channel := params[0]
		if misc.IsFirstRune(channel, '@') {
			ref = telegram.NewChannelRef(channel)
		}
	}

	admins, err := front.checkAccess(front.user, ref)
	if err != nil {
		front.bot.SendText(front.chat, "Access denied: %s", err)
		return
	}

	if err := front.back.UnsubscribeAll(ref); err != nil {
		front.bot.SendText(front.chat, err.Error())
		return
	}

	front.bot.NotifyAll(admins, "#info\nSubscriptions cleared.\nChat: %s", ref)
}

func (front *T) unsubscribe(params []string) {
	if len(params) == 0 {
		front.bot.SendText(front.chat, "Invalid command.")
		return
	}

	thread, _, err := front.parseThread(params[0])
	if err != nil {
		front.bot.SendText(front.chat, "Invalid command: %s", err)
		return
	}

	ref := front.chat
	if len(params) > 1 {
		channel := params[1]
		if misc.IsFirstRune(channel, '@') {
			ref = telegram.NewChannelRef(params[1])
		}
	}

	admins, err := front.checkAccess(front.user, ref)
	if err != nil {
		front.bot.SendText(front.chat, "Access denied: %s", err)
		return
	}

	if err := front.back.Unsubscribe(ref, *thread); err != nil {
		front.bot.SendText(front.chat, err.Error())
		return
	}

	front.bot.NotifyAll(admins,
		"#info\nSubscription stopped.\nChat: %s\nThread: %s",
		ref, (*thread).URL())
}

func (front *T) subscribe(params []string) {
	if len(params) == 0 {
		front.bot.SendText(front.chat, "Invalid command.")
		return
	}

	thread, offset, err := front.parseThread(params[0])
	if err != nil {
		front.bot.SendText(front.chat, "Invalid command: %s", err)
		return
	}

	hash, err := front.getHash(*thread)
	if err != nil {
		front.bot.SendText(front.chat, "Unable to load thread title: %s", err)
		return
	}

	ref := front.chat
	if len(params) > 1 {
		channel := params[1]
		if misc.IsFirstRune(channel, '@') {
			ref = telegram.NewChannelRef(params[1])
		}
	}

	admins, err := front.checkAccess(front.user, ref)
	if err != nil {
		front.bot.SendText(front.chat, "Access denied: %s", err)
		return
	}

	if err := front.back.Subscribe(ref, *thread, hash, offset); err != nil {
		front.bot.SendText(front.chat, err.Error())
		return
	}

	front.bot.NotifyAll(admins,
		"#info\nSubscription OK.\nChat: %s\nThread: %s\nOffset: %d",
		ref, (*thread).URL(), offset)
}

func (front *T) status() {
	front.bot.SendText(front.chat, "Alive.")
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
