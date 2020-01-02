package main

import (
	"expvar"
	"os"
	"time"

	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	"github.com/jfk9w/hikkabot/mediator"
	"github.com/jfk9w/hikkabot/source"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/util"
)

func init() {
	launch := time.Now()
	expvar.NewString("launch").Set(launch.Format(time.RFC3339))
	expvar.Publish("uptime", expvar.Func(func() interface{} { return time.Now().Sub(launch).String() }))
}

func main() {
	config := new(struct {
		Aggregator struct {
			AdminID telegram.ID
			Aliases map[telegram.Username]telegram.ID
			Storage storage.SQLConfig
			Timeout string
		}
		Telegram struct {
			Username    string
			Token       string
			Proxy       string
			Concurrency int
		}
		Media  mediator.Config
		Reddit *reddit.Config
		Dvach  *struct{ Usercode string }
	})
	err := flu.Read(flu.File(os.Args[1]), util.YAML(config))
	if err != nil {
		panic(err)
	}
	timeout, err := time.ParseDuration(config.Aggregator.Timeout)
	if err != nil {
		panic(err)
	}
	telegram.SendDelays[telegram.PrivateChat] = time.Second
	bot := telegram.NewBot(flu.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		ProxyURL(config.Telegram.Proxy).
		NewClient(), config.Telegram.Token)
	mediator := mediator.New(config.Media)
	defer mediator.Shutdown()
	storage := storage.NewSQL(config.Aggregator.Storage)
	defer storage.Close()
	go bot.Send(config.Aggregator.AdminID, &telegram.Text{Text: "⬆️"}, nil)
	agg := &feed.Aggregator{
		Channel:  feed.Telegram{Client: bot.Client},
		Storage:  storage,
		Mediator: mediator,
		Timeout:  timeout,
		Aliases:  config.Aggregator.Aliases,
		AdminID:  config.Aggregator.AdminID,
	}
	if config.Dvach != nil {
		client := dvach.NewClient(nil, config.Dvach.Usercode)
		agg.AddSource(source.DvachCatalogSource{client}).
			AddSource(source.DvachThreadSource{client})
	}
	if config.Reddit != nil {
		client := reddit.NewClient(nil, *config.Reddit)
		agg.AddSource(source.RedditSource{client})
	}
	bot.Listen(config.Telegram.Concurrency, agg.Init().CommandListener(config.Telegram.Username))
}
