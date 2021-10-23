package main

import (
	"context"

	"github.com/jfk9w-go/flu"
	fluapp "github.com/jfk9w-go/flu/app"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"

	"hikkabot/app"
	"hikkabot/app/plugin"
)

var GitCommit = "dev"

func main() {
	fluapp.GormDialects["postgres"] = postgres.Open
	defer func() {
		if e := recover(); e != nil {
			logrus.Panic(e)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := app.Create(GitCommit, flu.DefaultClock)
	defer flu.CloseQuietly(app)

	app.ApplyConverterPlugins(plugin.Aconvert{"video/webm"})

	dvach := new(plugin.DvachClient)
	reddit := plugin.NewRedditClient(ctx)
	app.ApplyVendorPlugins(
		(*plugin.Subreddit)(reddit),
		(*plugin.DvachCatalog)(dvach),
		(*plugin.DvachThread)(dvach),
	)

	configurer := fluapp.DefaultConfigurer("hikkabot")
	fluapp.Run(ctx, app, configurer)

	flu.AwaitSignal()
}
