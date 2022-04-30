package core

import (
	"context"

	"hikkabot/core/internal/iface"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/ext/tapp"
)

type InterfaceConfig struct {
	SupervisorID telegram.ID            `yaml:"supervisorId" doc:"Telegram admin user ID. If not set, only public commands (e.g. /start) will be available."`
	Aliases      map[string]telegram.ID `yaml:"aliases,omitempty" doc:"Chat aliases to use in commands: keys are aliases and values are telegram IDs."`
}

type InterfaceContext interface {
	tapp.Context
	StorageContext
	PollerContext
	InterfaceConfig() InterfaceConfig
}

type Interface[C InterfaceContext] struct {
	*iface.Impl
}

func (i *Interface[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	var bot tapp.Mixin[C]
	if err := app.Use(ctx, &bot, false); err != nil {
		return err
	}

	var storage Storage[C]
	if err := app.Use(ctx, &storage, false); err != nil {
		return err
	}

	var poller Poller[C]
	if err := app.Use(ctx, &poller, false); err != nil {
		return err
	}

	config := app.Config().InterfaceConfig()
	if config.SupervisorID == 0 {
		logf.Get(i).Warnf(ctx, "telegram supervisor ID is not set – subscription management is disabled; "+
			"consider providing valid user ID (hint: use /start command to get it)")
	}

	i.Impl = &iface.Impl{
		Telegram:     bot.Bot(),
		Poller:       poller,
		Storage:      storage,
		SupervisorID: config.SupervisorID,
		Aliases:      config.Aliases,
	}

	return nil
}
