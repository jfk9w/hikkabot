package frontend

import (
	"strconv"
	"strings"

	"github.com/jfk9w-go/hikkabot/text"
	"github.com/jfk9w/hikkabot/dvach"
)

func (front *T) status(cmd Command) {
	cmd.reply("alive")
}

func (front *T) subscribe(cmd Command) {
	cmd.requireArity(1)

	chat := cmd.channelOrSelf(1)
	admins, ok := cmd.requireAdmin(chat)
	if !ok {
		return
	}

	ref, offset, err := front.ParseThread(cmd.param(0))
	if err != nil {
		cmd.reply("invalid command: %s", err)
		return
	}

	hashtag, err := front.Hashtag(ref)
	if err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	if err := front.back.Subscribe(chat, ref, hashtag, offset); err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	front.bot.NotifyAll(admins,
		"#info\nSubscription OK.\nChat: %s\nThread: %s\nOffset: %d",
		chat, dvach.FormatThreadURL(ref.Board, ref.NumString), offset)
}

func (front *T) unsubscribe(cmd Command) {
	if !cmd.requireArity(1) {
		return
	}

	chat := cmd.channelOrSelf(1)
	admins, ok := cmd.requireAdmin(chat)
	if !ok {
		return
	}

	ref, _, err := front.ParseThread(cmd.param(0))
	if err != nil {
		cmd.reply("invalid command: %s", err)
		return
	}

	if err := front.back.Unsubscribe(chat, ref); err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	front.bot.NotifyAll(admins,
		"#info\nSubscription stopped.\nChat: %s\nThread: %s",
		chat, text.FormatRef(ref))
}

func (front *T) unsubscribeAll(cmd Command) {
	ref := cmd.channelOrSelf(0)
	admins, ok := cmd.requireAdmin(ref)
	if !ok {
		return
	}

	if err := front.back.UnsubscribeAll(ref); err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	front.bot.NotifyAll(admins, "#info\nSubscriptions cleared.\nChat: %s", ref)
}

func (front *T) dump(cmd Command) {
	ref := cmd.channelOrSelf(0)
	if _, ok := cmd.requireAdmin(ref); !ok {
		return
	}

	entries, err := front.back.Dump(ref)
	if err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	if len(entries) == 0 {
		cmd.reply("empty")
		return
	}

	sb := &strings.Builder{}
	for ref, entry := range entries {
		sb.WriteString(text.FormatRef(ref))
		sb.WriteRune('\n')
		sb.WriteString(entry.Hashtag)
		sb.WriteString("\n\n")
	}

	cmd.reply(sb.String())
}

func (front *T) popular(cmd Command) {
	if !cmd.requireArity(1) {
		return
	}

	board := cmd.param(0)
	limit := 30
	if l, err := strconv.Atoi(cmd.param(1)); err == nil {
		limit = l
	}

	if limit == 0 {
		return
	}

	catalog, err := front.dvch.Catalog(board)
	if err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	front.bot.SendPopular(cmd.chat, catalog.Threads, limit)
}
