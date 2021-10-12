package main

import (
	"context"
	"os"

	"github.com/jfk9w-go/flu"
	"github.com/sirupsen/logrus"

	"hikkabot/app"
	"hikkabot/app/plugin"
)

var GitCommit = "dev"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := app.Create(GitCommit, flu.DefaultClock, flu.File(os.Args[1]))
	if err != nil {
		logrus.Fatalf("initialize app: %s", err)
	}

	defer flu.CloseQuietly(app)

	if err := app.ConfigureLogging(); err != nil {
		logrus.Fatalf("configure logging: %s", err)
	}

	defer func() {
		if e := recover(); e != nil {
			logrus.Panic(e)
		}
	}()

	dvach := new(plugin.DvachClient)
	reddit := plugin.NewRedditClient(ctx)
	app.ApplyConverterPlugins(plugin.Aconvert{"video/webm"})
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
