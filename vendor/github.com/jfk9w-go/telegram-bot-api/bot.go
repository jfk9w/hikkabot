package telegram

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/httpf"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
)

const rootLoggerName = "telegram.bot"

func log() logf.Interface {
	return logf.Get(rootLoggerName)
}

type Bot struct {
	*baseClient
	*floodControlAware
	*conversationAware
	ctx    context.Context
	cancel context.CancelFunc
	work   syncf.WaitGroup
	me     *User
	once   sync.Once
}

func NewBot(clock syncf.Clock, client httpf.Client, token string) *Bot {
	if token == "" {
		log().Panicf(nil, "token must not be empty")
	}

	if client == nil {
		transport := httpf.NewDefaultTransport()
		transport.ResponseHeaderTimeout = 2 * time.Minute
		client = &http.Client{Transport: transport}
	}

	baseClient := &baseClient{
		client:   client,
		endpoint: func(method string) string { return "https://api.telegram.org/bot" + token + "/" + method },
	}

	floodControlAware := &floodControlAware{
		clock:    clock,
		executor: baseClient,
	}

	conversationAware := &conversationAware{
		sender: floodControlAware,
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Bot{
		baseClient:        baseClient,
		floodControlAware: floodControlAware,
		conversationAware: conversationAware,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func (b *Bot) String() string {
	return rootLoggerName
}

func (b *Bot) Listen(options GetUpdatesOptions) <-chan Update {
	channel := make(chan Update)
	_, _ = syncf.GoWith(b.ctx, b.work.Spawn, func(ctx context.Context) {
		defer close(channel)
		for {
			updates, err := b.GetUpdates(ctx, options)
			switch {
			case syncf.IsContextRelated(err):
				return

			case err != nil:
				log().Warnf(ctx, "poll error: %s", err)
				if err := flu.Sleep(ctx, time.Duration(options.TimeoutSecs)*time.Second); err != nil {
					return
				}

			default:
				for _, update := range updates {
					log().Tracef(ctx, "received update %d", update.ID)
					if update.ID < options.Offset {
						continue
					}

					options.Offset = update.ID.Increment()
					if update.Message != nil && update.Message.ReplyToMessage != nil {
						if err := b.Answer(ctx, update.Message); err == nil {
							continue
						} else {
							log().Warnf(ctx, "answer %d: %s", update.Message.ID, err)
						}
					}

					if ctx.Err() != nil {
						return
					}

					select {
					case <-ctx.Done():
						return
					case channel <- update:
					}
				}
			}
		}
	})

	return channel
}

func (b *Bot) Username() Username {
	b.once.Do(func() {
		ctx, cancel := context.WithTimeout(b.ctx, time.Minute)
		defer cancel()
		if me, err := b.GetMe(ctx); err != nil {
			log().Panicf(ctx, "getMe failed: %v", err)
		} else {
			b.me = me
			logf.Get(b).Infof(ctx, "got username: %s", me.Username.String())
		}
	})

	return *b.me.Username
}

var DefaultCommandsOptions = &GetUpdatesOptions{
	TimeoutSecs:    60,
	AllowedUpdates: []string{"message", "edited_message", "callback_query"},
}

func (b *Bot) Commands() <-chan *Command {
	updates := b.Listen(*DefaultCommandsOptions)
	commands := make(chan *Command)
	_, _ = syncf.GoWith(b.ctx, b.work.Spawn, func(ctx context.Context) {
		defer close(commands)
		for update := range updates {
			if cmd := b.extractCommand(update); cmd != nil {
				commands <- cmd
			}
		}
	})

	return commands
}

func (b *Bot) CommandListener(value interface{}) *Bot {
	var listener CommandListener
	switch value := value.(type) {
	case CommandListener:
		listener = value
	default:
		registry := make(CommandRegistry)
		if err := registry.From(value); err != nil {
			logf.Get(b).Panicf(nil, "register command listener from %T: %v", value, err)
		}

		listener = registry
	}

	commands := b.Commands()
	_, _ = syncf.GoWith(b.ctx, b.work.Spawn, func(ctx context.Context) {
		for cmd := range commands {
			err := b.onStart(ctx, cmd)
			switch {
			case syncf.IsContextRelated(err):
				return
			case err == nil:
				err = listener.OnCommand(ctx, b, cmd)
				if syncf.IsContextRelated(err) {
					return
				}
			}

			logf.Get(b).Resultf(ctx, logf.Debug, logf.Error, "handle %s: %v", cmd, err)

			if err != nil {
				_ = cmd.Reply(ctx, b, err.Error())
			}
		}
	})

	return b
}

func (b *Bot) onStart(ctx context.Context, cmd *Command) error {
	if cmd.Key == "/start" && cmd.Payload != "" {
		var payload string
		if data, err := base64.URLEncoding.DecodeString(cmd.Payload); err == nil {
			payload = string(data)
		} else if data, err := url.QueryUnescape(cmd.Payload); err == nil {
			payload = data
		} else {
			return err
		}

		cmd.init(b.Username(), payload)
	}

	return nil
}

func (b *Bot) CommandListenerFunc(fun CommandListenerFunc) *Bot {
	return b.CommandListener(fun)
}

func (b *Bot) HandleCommand(ctx context.Context, listener CommandListener, cmd *Command) error {
	err := listener.OnCommand(ctx, b, cmd)
	if syncf.IsContextRelated(err) {
		return err
	}

	if err != nil {
		replyErr := cmd.Reply(ctx, b, err.Error())
		log().Resultf(ctx, logf.Debug, logf.Warn, "reply to %s with %s: %s", cmd, err, replyErr)
		if syncf.IsContextRelated(replyErr) {
			return replyErr
		}
	}

	return nil
}

func (b *Bot) extractCommand(update Update) *Command {
	switch {
	case update.Message != nil:
		return b.extractCommandMessage(update.Message)
	case update.EditedMessage != nil:
		return b.extractCommandMessage(update.EditedMessage)
	case update.CallbackQuery != nil:
		return b.extractCommandCallbackQuery(update.CallbackQuery)
	default:
		return nil
	}
}

func (b *Bot) extractCommandMessage(message *Message) *Command {
	for _, entity := range message.Entities {
		if entity.Type == "bot_command" {
			cmd := &Command{
				User:    &message.From,
				Chat:    &message.Chat,
				Message: message,
			}

			cmd.init(b.Username(), message.Text[entity.Offset:])
			return cmd
		}
	}

	return nil
}

func (b *Bot) extractCommandCallbackQuery(query *CallbackQuery) *Command {
	if query.Data == nil {
		return nil
	}

	cmd := &Command{
		Chat:            &query.Message.Chat,
		User:            &query.From,
		Message:         query.Message,
		CallbackQueryID: query.ID,
	}

	cmd.init(b.Username(), *query.Data)
	return cmd
}

func trim(value string) string {
	return strings.Trim(value, " \n\t\v")
}

func (b *Bot) Close() error {
	b.cancel()
	b.work.Wait()
	return nil
}
