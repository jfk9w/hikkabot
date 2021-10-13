package main

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"

	"hikkabot/app"
	"hikkabot/app/plugin"
)

var GitCommit = "dev"

func main() {
	app.GormDialects["postgres"] = postgres.Open
	app, err := app.Create(GitCommit, flu.DefaultClock)
	if err != nil {
		logrus.Fatalf("initialize app: %s", err)
	}

	defer flu.CloseQuietly(app)

	if ok, err := app.Aux(); ok {
		return
	} else if err != nil {
		logrus.Fatalf("process aux command: %s", err)
	}

	if err := app.ConfigureLogging(); err != nil {
		logrus.Fatalf("configure logging: %s", err)
	}

	defer func() {
		if e := recover(); e != nil {
			logrus.Panic(e)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.ApplyConverterPlugins(plugin.Aconvert{"video/webm"})

	dvach := new(plugin.DvachClient)
	reddit := plugin.NewRedditClient(ctx)
	app.ApplyVendorPlugins(
		(*plugin.Subreddit)(reddit),
		(*plugin.DvachCatalog)(dvach),
		(*plugin.DvachThread)(dvach),
	)

	if err := app.Run(ctx); err != nil {
		logrus.Fatalf("run app: %s", err)
	}

	flu.AwaitSignal()
}
