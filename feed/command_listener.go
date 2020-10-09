package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/jfk9w-go/flu/metrics"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type WriteHTMLWithChatLink func(html *format.HTMLWriter, chatLink string) *format.HTMLWriter

type Management interface {
	CheckAccess(ctx context.Context, userID telegram.ID, chatID telegram.ID) (context.Context, error)
	NotifyAdmins(ctx context.Context, chatID telegram.ID, markup telegram.ReplyMarkup, writeHTML WriteHTMLWithChatLink) error
	GetChatLink(ctx context.Context, chatID telegram.ID) string
}

type Supervisor struct {
	client      telegram.Client
	userIDs     map[telegram.ID]bool
	inviteLinks map[telegram.ID]string
	flu.RWMutex
}

func NewSupervisorManagement(client telegram.Client, userIDs ...telegram.ID) *Supervisor {
	s := &Supervisor{
		client:      client,
		userIDs:     make(map[telegram.ID]bool),
		inviteLinks: make(map[telegram.ID]string),
	}

	for _, userID := range userIDs {
		s.userIDs[userID] = true
	}

	return s
}

func (s *Supervisor) GetChatLink(ctx context.Context, chatID telegram.ID) string {
	if chatID > 0 {
		return "tg://resolve?domain=" + s.client.Username()
	} else {
		s.RLock()
		if inviteLink, ok := s.inviteLinks[chatID]; ok {
			s.RUnlock()
			return inviteLink
		}

		s.RUnlock()
		defer s.Lock().Unlock()
		chat, err := s.client.GetChat(ctx, chatID)
		if err != nil {
			return fmt.Sprintf("getchat:%s", err)
		}

		inviteLink := chat.InviteLink
		if inviteLink == "" {
			if chat.Username != nil {
				inviteLink = "https://t.me/" + chat.Username.String()
			} else {
				inviteLink, err = s.client.ExportChatInviteLink(ctx, chatID)
				if err != nil {
					return fmt.Sprintf("exportchatinvitelink:%s", err)
				}
			}
		}

		s.inviteLinks[chatID] = inviteLink
		return inviteLink
	}
}

func (s *Supervisor) CheckAccess(ctx context.Context, userID telegram.ID, _ telegram.ID) (context.Context, error) {
	if _, ok := s.userIDs[userID]; !ok {
		return nil, ErrForbidden
	} else {
		return ctx, nil
	}
}

func (s *Supervisor) NotifyAdmins(ctx context.Context, chatID telegram.ID, markup telegram.ReplyMarkup, writeHTML WriteHTMLWithChatLink) error {
	chatLink := s.GetChatLink(ctx, chatID)
	transport := format.NewBufferTransport()
	if err := writeHTML(format.HTMLWithTransport(ctx, transport), chatLink).Flush(); err != nil {
		return errors.Wrap(err, "flush")
	}

	lastIdx := len(transport.Pages) - 1
	pages := transport.Pages
	userIDs := make([]telegram.ChatID, len(s.userIDs))
	i := 0
	for userID, _ := range s.userIDs {
		userIDs[i] = userID
		i++
	}

	ttransport := &format.TelegramTransport{
		Sender:  s.client,
		ChatIDs: userIDs,
		Strict:  true,
	}

	for i, page := range pages {
		markup := markup
		if i != lastIdx {
			markup = nil
		}

		if err := ttransport.Text(
			format.WithParseMode(format.WithReplyMarkup(ctx, markup), telegram.HTML),
			page, true); err != nil {
			return err
		}
	}

	return nil
}

type CommandListener struct {
	Context    context.Context
	Aggregator *Aggregator
	Management Management
	Aliases    map[string]telegram.ID
	Metrics    metrics.Registry
}

func (c *CommandListener) Init(ctx context.Context) (*CommandListener, error) {
	if c.Metrics == nil {
		c.Metrics = metrics.DummyRegistry{}
	}

	return c, c.Aggregator.Init(ctx, c)
}

func (c *CommandListener) Close() error {
	return c.Aggregator.Close()
}

const (
	suspendCommandKey = "s"
	resumeCommandKey  = "r"
	deleteCommandKey  = "d"
)

func (c *CommandListener) OnCommand(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	var fun func(context.Context, telegram.Client, telegram.Command) error
	c.Metrics.Counter(cmd.Key, metrics.Labels{
		"chat_id", cmd.Chat.ID.String(),
		"user_id", cmd.User.ID.String(),
	}).Inc()
	switch cmd.Key {
	case "/sub", "/subscribe":
		fun = c.Subscribe
	case suspendCommandKey:
		fun = c.Suspend
	case resumeCommandKey:
		fun = c.Resume
	case deleteCommandKey:
		fun = c.Delete
	case "/clear":
		fun = c.Clear
	case "/list":
		fun = c.List
	case "/status":
		fun = c.Status
	default:
		return errors.New("invalid command")
	}

	if err := fun(ctx, client, cmd); err != nil {
		return err
	}
	if len(cmd.Key) == 1 {
		// callback query
		return cmd.Reply(ctx, client, "OK")
	}
	return nil
}

func (c *CommandListener) background() (context.Context, func()) {
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithTimeout(ctx, time.Minute)
}

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

func (c *CommandListener) resolveChatID(ctx context.Context, client telegram.Client, cmd telegram.Command, argumentIndex int) (context.Context, telegram.ID, error) {
	chatID := cmd.Chat.ID
	if len(cmd.Args) > argumentIndex && cmd.Args[argumentIndex] != "." {
		if id, ok := c.Aliases[cmd.Args[argumentIndex]]; ok {
			chatID = id
		} else {
			chat, err := client.GetChat(ctx, telegram.Username(cmd.Args[argumentIndex]))
			if err != nil {
				chatID, err = telegram.ParseID(cmd.Args[argumentIndex])
				if err != nil {
					return nil, 0, errors.Wrap(err, "parse chat ID")
				}
			} else {
				chatID = chat.ID
			}
		}
	}

	ctx, err := c.Management.CheckAccess(ctx, cmd.User.ID, chatID)
	if err != nil {
		return nil, 0, err
	}

	return ctx, chatID, nil
}

func (c *CommandListener) parseSubID(ctx context.Context, cmd telegram.Command, argumentIndex int) (context.Context, SubID, error) {
	subID, err := ParseSubID(cmd.Args[argumentIndex])
	if err != nil {
		return nil, SubID{}, errors.Wrap(err, "parse subID")
	}
	ctx, err = c.Management.CheckAccess(ctx, cmd.User.ID, telegram.ID(subID.FeedID))
	if err != nil {
		return nil, SubID{}, err
	}
	return ctx, subID, nil
}

func (c *CommandListener) Subscribe(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	if len(cmd.Args) == 0 {
		return ErrSubscribeUsage
	}
	ref := cmd.Args[0]
	ctx, chatID, err := c.resolveChatID(ctx, client, cmd, 1)
	if err != nil {
		return err
	}
	var options []string
	if len(cmd.Args) > 2 {
		options = cmd.Args[2:]
	}

	sub, err := c.Aggregator.Subscribe(ctx, ID(chatID), ref, options)
	if err != nil {
		return err
	}
	go c.OnSubscribe(sub)
	return nil
}

func (c *CommandListener) OnSubscribe(sub Sub) {
	ctx, cancel := c.background()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID),
		telegram.InlineKeyboard([]telegram.Button{
			telegram.Command{Key: suspendCommandKey, Args: []string{sub.SubID.String()}}.Button("Suspend"),
		}),
		func(html *format.HTMLWriter, chatLink string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatLink).
				Text(" 🔥")
		})
}

func (c *CommandListener) Suspend(ctx context.Context, _ telegram.Client, cmd telegram.Command) error {
	ctx, subID, err := c.parseSubID(ctx, cmd, 0)
	if err != nil {
		return err
	}
	sub, err := c.Aggregator.Suspend(ctx, subID, ErrSuspendedByUser)
	if err != nil {
		return err
	}
	go c.OnSuspend(sub, ErrSuspendedByUser)
	return nil
}

func (c *CommandListener) OnSuspend(sub Sub, err error) {
	ctx, cancel := c.background()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID),
		// by column
		telegram.InlineKeyboard([]telegram.Button{
			telegram.Command{Key: resumeCommandKey, Args: []string{sub.SubID.String()}}.Button("Resume"),
			telegram.Command{Key: deleteCommandKey, Args: []string{sub.SubID.String()}}.Button("Delete"),
		}),
		func(html *format.HTMLWriter, chatLink string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatLink).
				Text(" 🛑\n" + err.Error())
		})
}

func (c *CommandListener) Resume(ctx context.Context, _ telegram.Client, cmd telegram.Command) error {
	ctx, subID, err := c.parseSubID(ctx, cmd, 0)
	if err != nil {
		return err
	}
	sub, err := c.Aggregator.Resume(ctx, subID)
	if err != nil {
		return err
	}
	go c.OnResume(sub)
	return nil
}

func (c *CommandListener) OnResume(sub Sub) {
	ctx, cancel := c.background()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID),
		telegram.InlineKeyboard([]telegram.Button{
			telegram.Command{Key: suspendCommandKey, Args: []string{sub.SubID.String()}}.Button("Suspend"),
		}),
		func(html *format.HTMLWriter, chatLink string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatLink).
				Text(" 🔥")
		})
}

func (c *CommandListener) Delete(ctx context.Context, _ telegram.Client, cmd telegram.Command) error {
	ctx, subID, err := c.parseSubID(ctx, cmd, 0)
	if err != nil {
		return err
	}
	sub, err := c.Aggregator.Delete(ctx, subID)
	if err != nil {
		return err
	}
	go c.OnDelete(sub)
	return nil
}

func (c *CommandListener) OnDelete(sub Sub) {
	ctx, cancel := c.background()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(sub.FeedID), nil,
		func(html *format.HTMLWriter, chatLink string) *format.HTMLWriter {
			return html.Text(sub.Name+" @ ").
				Link("chat", chatLink).
				Text(" 🗑")
		})
}

func (c *CommandListener) Clear(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	if len(cmd.Args) != 2 {
		return ErrClearUsage
	}
	ctx, chatID, err := c.resolveChatID(ctx, client, cmd, 1)
	if err != nil {
		return err
	}
	pattern := cmd.Args[0]
	count, err := c.Aggregator.Clear(ctx, ID(chatID), pattern)
	if err != nil {
		return err
	}
	go c.OnClear(ID(chatID), pattern, count)
	return nil
}

func (c *CommandListener) OnClear(feedID ID, pattern string, count int64) {
	ctx, cancel := c.background()
	defer cancel()
	_ = c.Management.NotifyAdmins(ctx, telegram.ID(feedID), nil,
		func(html *format.HTMLWriter, chatLink string) *format.HTMLWriter {
			return html.Text(fmt.Sprintf("%d subs @ ", count)).
				Link("chat", chatLink).
				Text(" 🗑 (" + pattern + ")")
		})
}

func (c *CommandListener) List(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	ctx, chatID, err := c.resolveChatID(ctx, client, cmd, 0)
	if err != nil {
		return err
	}

	active := len(cmd.Args) > 1 && cmd.Args[1] == resumeCommandKey
	subs, err := c.Aggregator.List(ctx, ID(chatID), active)
	if err != nil {
		return err
	}
	if !active && len(subs) == 0 {
		active = true
		subs, err = c.Aggregator.List(ctx, ID(chatID), active)
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
		keyboard[i] = []telegram.Button{telegram.Command{Key: changeCmd, Args: []string{sub.SubID.String()}}.Button(sub.Name)}
	}

	_, err = client.Send(ctx, cmd.Chat.ID,
		telegram.Text{
			ParseMode: telegram.HTML,
			Text: fmt.Sprintf(
				"%d subs @ %s %s",
				len(subs),
				format.HTMLAnchor("chat", c.Management.GetChatLink(ctx, chatID)),
				status),
			DisableWebPagePreview: true},
		&telegram.SendOptions{
			ReplyMarkup:      telegram.InlineKeyboard(keyboard...),
			ReplyToMessageID: cmd.Message.ID})
	return err
}

func (c *CommandListener) Status(ctx context.Context, client telegram.Client, cmd telegram.Command) error {
	return cmd.Reply(ctx, client, fmt.Sprintf("OK\n"+
		"User ID: %s\n"+
		"Chat ID: %s\n"+
		"Message ID: %s\n"+
		"Bot username: %s\n"+
		"Datetime: %s\n",
		cmd.User.ID, cmd.Chat.ID, cmd.Message.ID,
		client.Username(), time.Now().Format("2006-01-02 15:04:05")))
}
