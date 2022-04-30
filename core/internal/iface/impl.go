package iface

import (
	"context"
	"fmt"
	"strings"

	"hikkabot/feed"

	"github.com/jfk9w-go/flu/colf"

	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext"
	"github.com/jfk9w-go/telegram-bot-api/ext/receiver"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
	"github.com/pkg/errors"
)

const (
	suspend = "s"
	resume  = "r"
	delete  = "d"

	fire     = "🔥"
	stop     = "🛑"
	bin      = "🗑"
	thumbsUp = "👍"
)

type Impl struct {
	Telegram     telegram.Client
	Poller       feed.Poller
	Storage      feed.Storage
	SupervisorID telegram.ID
	Aliases      map[string]telegram.ID
}

func (i *Impl) String() string {
	return ServiceID
}

func (i *Impl) CommandScope() tapp.CommandScope {
	if i.SupervisorID == 0 {
		return tapp.CommandScope{}
	}

	return tapp.CommandScope{
		UserIDs: colf.Set[telegram.ID]{i.SupervisorID: true},
	}
}

//
// Command listeners
//

func (i *Impl) Subscribe(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) == 0 {
		return errSubscribe
	}

	ref := cmd.Args[0]
	ctx, feedID, err := i.resolveFeedID(ctx, cmd, 1)
	if err != nil {
		return err
	}

	var options []string
	if len(cmd.Args) > 2 {
		options = cmd.Args[2:]
	}

	if err := i.Poller.Subscribe(ctx, feedID, ref, options); err != nil {
		return err
	}

	return cmd.ReplyCallback(ctx, i.Telegram, thumbsUp)
}

func (i *Impl) Suspend(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	header, err := i.parseHeader(cmd, 0)
	if err != nil {
		return err
	}

	return i.Poller.Suspend(ctx, header, feed.ErrSuspendedByUser)
}

func (i *Impl) Resume(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	header, err := i.parseHeader(cmd, 0)
	if err != nil {
		return err
	}

	return i.Poller.Resume(ctx, header)
}

func (i *Impl) Delete(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	header, err := i.parseHeader(cmd, 0)
	if err != nil {
		return err
	}

	return i.Poller.Delete(ctx, header)
}

func (i *Impl) Clear(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 1 {
		return errDeleteAll
	}

	ctx, feedID, err := i.resolveFeedID(ctx, cmd, 1)
	if err != nil {
		return err
	}

	pattern := cmd.Args[0]
	return i.Poller.Clear(ctx, feedID, pattern)
}

func (i *Impl) List(ctx context.Context, _ telegram.Client, cmd *telegram.Command) error {
	ctx, feedID, err := i.resolveFeedID(ctx, cmd, 0)
	if err != nil {
		return err
	}

	active := len(cmd.Args) > 1 && cmd.Args[1] == resume
	subs, err := i.Storage.ListSubscriptions(ctx, feedID, active)
	if err != nil {
		return err
	}

	if !active && len(subs) == 0 {
		active = true
		subs, err = i.Storage.ListSubscriptions(ctx, feedID, active)
		if err != nil {
			return err
		}
	}

	status, changeCmd := stop, resume
	if active {
		status, changeCmd = fire, suspend
	}

	// by row
	keyboard := make([][]telegram.Button, len(subs))
	for i, sub := range subs {
		keyboard[i] = []telegram.Button{
			(&telegram.Command{Key: changeCmd, Args: []string{formatHeader(sub.Header)}}).Button(sub.Name),
		}
	}

	text := telegram.Text{
		ParseMode:             telegram.HTML,
		Text:                  fmt.Sprintf("%d subs %s", len(subs), status),
		DisableWebPagePreview: true,
	}

	_, err = i.Telegram.Send(ctx, cmd.Chat.ID, text, &telegram.SendOptions{
		ReplyMarkup:      telegram.InlineKeyboard(keyboard...),
		ReplyToMessageID: cmd.Message.ID,
	})

	return err
}

func (i *Impl) Sub(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	return i.Subscribe(ctx, client, cmd)
}

//
// Callback aliases
//

func (i *Impl) S_callback(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if err := i.Suspend(ctx, client, cmd); err != nil {
		return err
	}

	return cmd.ReplyCallback(ctx, client, thumbsUp)
}

func (i *Impl) R_callback(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if err := i.Resume(ctx, client, cmd); err != nil {
		return err
	}

	return cmd.ReplyCallback(ctx, client, thumbsUp)
}

func (i *Impl) D_callback(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if err := i.Delete(ctx, client, cmd); err != nil {
		return err
	}

	return cmd.ReplyCallback(ctx, client, thumbsUp)
}

//
// After triggers
//

func (i *Impl) AfterResume(ctx context.Context, sub *feed.Subscription) error {
	chatTitle, err := i.getChatTitle(ctx, sub.FeedID)
	if err != nil {
		return err
	}

	buttons := []telegram.Button{
		(&telegram.Command{Key: suspend, Args: []string{formatHeader(sub.Header)}}).Button("Suspend"),
	}

	ctx = receiver.ReplyMarkup(ctx, telegram.InlineKeyboard(buttons))
	return ext.HTML(ctx, i.Telegram, i.SupervisorID).
		Text(sub.Name+" @ ").
		Text(chatTitle).
		Text(" %s", fire).
		Flush()
}

func (i *Impl) AfterSuspend(ctx context.Context, sub *feed.Subscription) error {
	chatTitle, err := i.getChatTitle(ctx, sub.FeedID)
	if err != nil {
		return err
	}

	buttons := []telegram.Button{
		(&telegram.Command{Key: resume, Args: []string{formatHeader(sub.Header)}}).Button("Resume"),
		(&telegram.Command{Key: delete, Args: []string{formatHeader(sub.Header)}}).Button("Delete"),
	}

	ctx = receiver.ReplyMarkup(ctx, telegram.InlineKeyboard(buttons))
	return ext.HTML(ctx, i.Telegram, i.SupervisorID).
		Text(sub.Name+" @ ").
		Text(chatTitle).
		Text(" %s\n%s", stop, sub.Error.String).
		Flush()
}

func (i *Impl) AfterDelete(ctx context.Context, sub *feed.Subscription) error {
	chatTitle, err := i.getChatTitle(ctx, sub.FeedID)
	if err != nil {
		return err
	}

	return ext.HTML(ctx, i.Telegram, i.SupervisorID).
		Text(sub.Name+" @ ").
		Text(chatTitle).
		Text(" %s", bin).
		Flush()
}

func (i *Impl) AfterClear(ctx context.Context, feedID feed.ID, pattern string, deleted int64) error {
	chatTitle, err := i.getChatTitle(ctx, feedID)
	if err != nil {
		return err
	}

	return ext.HTML(ctx, i.Telegram, i.SupervisorID).
		Text(fmt.Sprintf("%d subs @ ", deleted)).
		Text(chatTitle).
		Text(" %s (%s)", bin, pattern).
		Flush()
}

//
// Implementation details
//

const headerDelimiter = "+"

func formatHeader(header feed.Header) string {
	return strings.Join([]string{header.FeedID.String(), header.Vendor, header.SubID}, headerDelimiter)
}

func (i *Impl) parseHeader(cmd *telegram.Command, argumentIndex int) (header feed.Header, err error) {
	arg := cmd.Args[argumentIndex]
	tokens := strings.Split(arg, headerDelimiter)
	if len(tokens) != 3 {
		err = errors.Errorf("invalid header [%s]", header)
		return
	}

	feedID, err := telegram.ParseID(tokens[0])
	if err != nil {
		err = errors.Wrapf(err, "invalid string id: %s", tokens[2])
		return
	}

	header.SubID = tokens[2]
	header.Vendor = tokens[1]
	header.FeedID = feed.ID(feedID)

	return
}

type contextValues struct {
	feed *telegram.Chat
}

func (i *Impl) getChatTitle(ctx context.Context, feedID feed.ID) (string, error) {
	var feed *telegram.Chat
	if values, ok := ctx.Value(contextValues{}).(contextValues); ok && values.feed != nil {
		feed = values.feed
	} else {
		var err error
		feed, err = i.Telegram.GetChat(ctx, telegram.ID(feedID))
		switch {
		case syncf.IsContextRelated(err):
			return "", err
		case err != nil:
			logf.Get(i).Errorf(ctx, "get chat %d: %v", feedID, err)
			return fmt.Sprint(feedID), nil
		}
	}

	if feed.Type == telegram.PrivateChat {
		return "<private>", nil
	}

	return feed.Title, nil
}

func (i *Impl) resolveFeedID(ctx context.Context, cmd *telegram.Command, argumentIndex int) (context.Context, feed.ID, error) {
	chatID := cmd.Chat.ID
	if len(cmd.Args) > argumentIndex {
		if arg := cmd.Args[argumentIndex]; arg != "." {
			if id, ok := i.Aliases[arg]; ok {
				chatID = id
			} else {
				chat, err := i.Telegram.GetChat(ctx, telegram.Username(arg))
				if err != nil {
					logf.Get(i).Warnf(ctx, "failed to get chat %s: %v", arg, err)
					chatID, err = telegram.ParseID(arg)
					if err != nil {
						return nil, 0, errors.Wrap(err, "parse header")
					}
				} else {
					chatID = chat.ID
					ctx = context.WithValue(ctx, contextValues{}, contextValues{
						feed: chat,
					})
				}
			}
		}
	}

	return ctx, feed.ID(chatID), nil
}
