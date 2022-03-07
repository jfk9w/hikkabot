package main

import (
	"context"

	"github.com/jfk9w-go/flu"
	apfel "github.com/jfk9w-go/flu/apfel"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"

	"hikkabot/app"
	"hikkabot/app/plugin"
)

var GitCommit = "dev"

func main() {
	apfel.GormDialects["postgres"] = postgres.Open
	defer func() {
		if e := recover(); e != nil {
			logrus.Panic(e)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := app.Create(GitCommit, flu.DefaultClock)
	defer flu.CloseQuietly(app)

	app.ApplyConverterPlugins(plugin.FFmpeg, plugin.Aconvert)

	dvach := new(plugin.DvachClient)
	reddit := plugin.NewRedditClient(ctx)
	app.ApplyVendorPlugins(
		(*plugin.Subreddit)(reddit),
		(*plugin.DvachCatalog)(dvach),
		(*plugin.DvachThread)(dvach),
	)

	configurer := apfel.DefaultConfigurer("hikkabot")
	apfel.Run(ctx, app, configurer)
}
