package frontend

import (
	"strconv"
	"strings"

	"github.com/jfk9w-go/hikkabot/html"
	"github.com/jfk9w/hikkabot/util"
)

func (front *T) status(cmd command) {
	cmd.reply("alive")
}

func (front *T) subscribe(cmd command) {
	cmd.requireArity(1)

	thread, offset, err := front.parseThread(cmd.param(0))
	if err != nil {
		cmd.reply("invalid command: %s", err)
		return
	}

	hash, err := front.hashify(*thread)
	if err != nil {
		cmd.reply("unable to load thread title: %s", err)
		return
	}

	ref := cmd.channelOrSelf(1)
	admins, ok := cmd.requireAdmin(ref)
	if !ok {
		return
	}

	if err := front.back.Subscribe(ref, *thread, hash, offset); err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	front.bot.NotifyAll(admins,
		"#info\nSubscription OK.\nChat: %s\nThread: %s\nOffset: %d",
		ref, (*thread).URL(), offset)
}

func (front *T) unsubscribe(cmd command) {
	if !cmd.requireArity(1) {
		return
	}

	thread, _, err := front.parseThread(cmd.param(0))
	if err != nil {
		cmd.reply("invalid command: %s", err)
		return
	}

	ref := cmd.channelOrSelf(1)
	admins, ok := cmd.requireAdmin(ref)
	if !ok {
		return
	}

	if err := front.back.Unsubscribe(ref, *thread); err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	front.bot.NotifyAll(admins,
		"#info\nSubscription stopped.\nChat: %s\nThread: %s",
		ref, (*thread).URL())
}

func (front *T) unsubscribeAll(cmd command) {
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

func (front *T) dump(cmd command) {
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
	for thread, entry := range entries {
		sb.WriteString(html.Num(thread.Board, thread.Num))
		sb.WriteRune('\n')
		sb.WriteString(entry.Hash)
		sb.WriteString("\n\n")
	}

	cmd.reply(sb.String())
}

func (front *T) catalog(cmd command) {
	if !cmd.requireArity(1) {
		return
	}

	board := cmd.param(0)
	limit := 30
	if l, err := strconv.Atoi(cmd.param(1)); err == nil {
		limit = l
	}

	catalog, err := front.dvch.Catalog(board)
	if err != nil {
		cmd.reply("failed: %s", err)
		return
	}

	posts := catalog.Threads
	limit = util.MinInt(len(posts), limit)
	for i := 0; i < limit; i++ {
		front.bot.SendPost(cmd.chat, html.Post{posts[i], board, ""}, true)
	}

	cmd.reply("FIN")
}
