package main

import (
	"context"
	"flag"
	"os"

	"github.com/jfk9w-go/flu"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"

	"hikkabot/app"
	"hikkabot/app/plugin"
)

var (
	dryRun = flag.Bool("dry-run", false, "Prints out the collected config.")
	stdin  = flag.Bool("stdin", false, "Accept config input from stdin.")
)

var GitCommit = "dev"

func main() {
	flag.Parse()

	config, err := config(flag.Args(), *stdin)
	if err != nil {
		logrus.Fatalf("get config: %s", err)
	}

	if *dryRun {
		println(config.Unmask().String())
		return
	}

	app.GormDialects["postgres"] = postgres.Open
	run(config)
}

func run(config flu.Input) {
	app, err := app.Create(GitCommit, flu.DefaultClock, config)
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

func config(args []string, stdin bool) (*flu.ByteBuffer, error) {
	configsLen := len(args)
	if stdin {
		configsLen += 1
	}

	configs := make([]flu.Input, configsLen)
	for i, arg := range args {
		configs[i] = flu.File(arg)
	}

	if stdin {
		configs[configsLen-1] = flu.IO{R: os.Stdin}
	}

	return app.CollectConfig("HIKKABOT_", configs...)
}
