package tapp

import (
	"context"
	"strings"
	"sync"

	"github.com/jfk9w-go/flu"
	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
	"github.com/jfk9w-go/telegram-bot-api"
)

type Config struct {
	Token string `yaml:"token" doc:"Telegram Bot API token."`
}

type Context interface {
	TelegramConfig() Config
}

type Listener interface {
	String() string
	Scoped
}

type Mixin[C Context] struct {
	version  string
	bot      *telegram.Bot
	commands Commands
	registry telegram.CommandRegistry
	once     sync.Once
}

func (m *Mixin[C]) String() string {
	return "telegram.bot"
}

func (m *Mixin[C]) Bot() *telegram.Bot {
	return m.bot
}

func (m *Mixin[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	m.version = app.Version()
	m.bot = telegram.NewBot(app, nil, app.Config().TelegramConfig().Token)
	m.commands = make(Commands)
	m.registry = make(telegram.CommandRegistry)
	return nil
}

func (m *Mixin[C]) AfterInclude(ctx context.Context, app apfel.MixinApp[C], mixin apfel.Mixin[C]) error {
	if _, ok := mixin.(Listener); !ok {
		return nil
	}

	local := make(telegram.CommandRegistry)
	if err := local.From(mixin); err != nil {
		logf.Get(m).Printf(ctx, "register %s error: %v", mixin, err)
		return nil
	}

	scope := Public
	if scoped, ok := mixin.(Scoped); ok {
		scope = scoped.CommandScope()
	}

	for key, listener := range local {
		scope.Transform(func(scope telegram.BotCommandScope) { m.commands.AddAll(scope, key) })
		m.registry.Add(key, scope.Wrap(listener))
		logf.Get(m).Infof(ctx, "register command %s @ [%s] for %s", key, mixin, scope)
	}

	return nil
}

func (m *Mixin[C]) Run(ctx context.Context) {
	defer logf.Get(m).Infof(ctx, "stopped")
	AddDefaultStart(m.commands, m.registry, m.version)
	if err := m.commands.Set(ctx, m.bot); err != nil {
		logf.Get(m).Warnf(ctx, "set commands: %v", err)
	}

	defer flu.CloseQuietly(m.bot.CommandListener(m.registry))
	logf.Get(m).Infof(ctx, "started")
	syncf.AwaitSignal(ctx)
}

func humanizeKey(key string) string {
	return strings.Replace(strings.Title(strings.Trim(key, "/")), "_", " ", -1)
}
