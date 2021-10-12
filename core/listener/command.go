package listener

import (
	"context"
	"fmt"

	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/html"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"hikkabot/core/feed"
)

var (
	ErrSubscribeUsage = errors.Errorf("" +
		"Usage: /subscribe SUB [CHAT_ID] [OPTIONS]\n\n" +
		"SUB – subscription string (for example, a link).\n" +
		"CHAT_ID – target chat username or '.' to use this chat. Optional, this chat by default.\n" +
		"OPTIONS – subscription-specific options string. Optional, empty by default.")

	ErrClearUsage = errors.Errorf("" +
		"Usage: /clear PATTERN [CHAT_ID]\n\n" +
		"PATTERN – pattern to match subscription error.\n" +
		"CHAT_ID – target chat username or '.' to use this chat.",
	)

	ErrListUsage = errors.Errorf("" +
		"Usage: /list [CHAT_ID] [STATUS]\n\n" +
		"CHAT_ID – target chat username or '.' to use this chat. Optional, this chat by default.\n" +
		"STATUS – status subscriptions to list for, 's' for suspended. Optional, active by default.")
)

const (
	suspendCommandKey = "s"
	resumeCommandKey  = "r"
	deleteCommandKey  = "d"
)

type Command struct {
	AccessControl
	Aggregator Aggregator
	Aliases    map[string]telegram.ID
	Version    string
	Vendors    []Vendor
}

func (l *Command) OnCommand(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	var fun telegram.CommandListenerFunc
	switch cmd.Key {
	case "/sub", "/subscribe":
		fun = l.Subscribe
	case suspendCommandKey:
		fun = l.Suspend
	case resumeCommandKey:
		fun = l.Resume
	case deleteCommandKey:
		fun = l.Delete
	case "/clear":
		fun = l.Clear
	case "/list":
		fun = l.List
	case "/status":
		fun = l.Status
	}

	if fun != nil {
		if err := fun(ctx, client, cmd); err != nil {
			return err
		}

		if cmd.CallbackQueryID != "" {
			return cmd.Reply(ctx, client, "OK")
		}

		return nil
	}

	for _, vendor := range l.Vendors {
		if ok, err := vendor.OnCommand(ctx, client, cmd); ok {
			if err != nil {
				return cmd.Reply(ctx, client, err.Error())
			}

			if cmd.CallbackQueryID != "" {
				return cmd.Reply(ctx, client, "OK")
			}

			return nil
		}
	}

	return nil
}

func (l *Command) Subscribe(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) == 0 {
		return ErrSubscribeUsage
	}

	ref := cmd.Args[0]
	ctx, feedID, err := l.resolveFeedID(ctx, client, cmd, 1)
	if err != nil {
		return err
	}
	var options []string
	if len(cmd.Args) > 2 {
		options = cmd.Args[2:]
	}

	return l.Aggregator.Subscribe(ctx, feedID, ref, options)
}

func (l *Command) Suspend(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	ctx, header, err := l.parseHeader(ctx, client, cmd, 0)
	if err != nil {
		return err
	}
	return l.Aggregator.Suspend(ctx, header, feed.ErrSuspendedByUser)
}

func (l *Command) Resume(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	ctx, header, err := l.parseHeader(ctx, client, cmd, 0)
	if err != nil {
		return err
	}
	return l.Aggregator.Resume(ctx, header)
}

func (l *Command) Delete(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	ctx, header, err := l.parseHeader(ctx, client, cmd, 0)
	if err != nil {
		return err
	}
	return l.Aggregator.Delete(ctx, header)
}

func (l *Command) Clear(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	if len(cmd.Args) < 1 {
		return ErrClearUsage
	}

	ctx, feedID, err := l.resolveFeedID(ctx, client, cmd, 1)
	if err != nil {
		return err
	}
	pattern := cmd.Args[0]
	return l.Aggregator.Clear(ctx, feedID, pattern)
}

func (l *Command) List(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	ctx, feedID, err := l.resolveFeedID(ctx, client, cmd, 0)
	if err != nil {
		return err
	}

	active := len(cmd.Args) > 1 && cmd.Args[1] == resumeCommandKey
	subs, err := l.Aggregator.List(ctx, feedID, active)
	if err != nil {
		return err
	}
	if !active && len(subs) == 0 {
		active = true
		subs, err = l.Aggregator.List(ctx, feedID, active)
		if err != nil {
			return err
		}
	}

	status, changeCmd := "🛑", resumeCommandKey
	if active {
		status, changeCmd = "🔥", suspendCommandKey
	}

	// by row
	keyboard := make([][]telegram.Button, len(subs))
	for i, sub := range subs {
		keyboard[i] = []telegram.Button{
			(&telegram.Command{Key: changeCmd, Args: []string{sub.Header.String()}}).Button(sub.Name),
		}
	}

	chatLink, err := l.GetChatLink(ctx, client, feedID)
	if err != nil {
		logrus.WithField("chat_id", feedID).
			Warnf("get chat link: %s", err)
		chatLink = feedID.String()
	} else {
		chatLink = html.Anchor("chat", chatLink)
	}

	_, err = client.Send(ctx, cmd.Chat.ID,
		telegram.Text{
			ParseMode:             telegram.HTML,
			Text:                  fmt.Sprintf("%d subs @ %s %s", len(subs), chatLink, status),
			DisableWebPagePreview: true},
		&telegram.SendOptions{
			ReplyMarkup:      telegram.InlineKeyboard(keyboard...),
			ReplyToMessageID: cmd.Message.ID})
	return err
}

func (l *Command) Status(ctx context.Context, client telegram.Client, cmd *telegram.Command) error {
	return cmd.Reply(ctx, client, fmt.Sprintf("OK\n"+
		"User ID: %s\n"+
		"Chat ID: %s\n"+
		"Username: %s\n"+
		"Version: %s\n",
		cmd.User.ID, cmd.Chat.ID, client.Username(), l.Version))
}

func (l *Command) parseHeader(ctx context.Context,
	client telegram.Client, cmd *telegram.Command, argumentIndex int) (
	context.Context, *feed.Header, error) {

	header, err := feed.ParseHeader(cmd.Args[argumentIndex])
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse header")
	}

	ctx, err = l.CheckAccess(ctx, client, cmd.User.ID, header.FeedID)
	if err != nil {
		return nil, nil, err
	}

	return ctx, header, nil
}

func (l *Command) resolveFeedID(ctx context.Context,
	client telegram.Client, cmd *telegram.Command, argumentIndex int) (
	context.Context, telegram.ID, error) {

	chatID := cmd.Chat.ID
	if len(cmd.Args) > argumentIndex {
		if arg := cmd.Args[argumentIndex]; arg != "." {
			if id, ok := l.Aliases[arg]; ok {
				chatID = id
			} else {
				chat, err := client.GetChat(ctx, telegram.Username(arg))
				if err != nil {
					logrus.WithField("chat", arg).Warnf("get chat: %s", err)
					chatID, err = telegram.ParseID(arg)
					if err != nil {
						return nil, 0, errors.Wrap(err, "parse header")
					}
				} else {
					chatID = chat.ID
				}
			}
		}
	}

	ctx, err := l.CheckAccess(ctx, client, cmd.User.ID, chatID)
	if err != nil {
		return nil, 0, err
	}

	return ctx, chatID, nil
}
