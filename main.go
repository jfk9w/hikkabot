package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	_aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	_mediator "github.com/jfk9w/hikkabot/mediator"
	_metrics "github.com/jfk9w/hikkabot/metrics"
	_source "github.com/jfk9w/hikkabot/source"
	_storage "github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/util"
	"github.com/pkg/errors"
)

type Config struct {
	Aggregator struct {
		AdminID telegram.ID
		Aliases map[telegram.Username]telegram.ID
		Storage _storage.SQLConfig
		Timeout string
	}
	Telegram struct {
		Username    string
		Token       string
		Proxy       string
		Concurrency int
		LogFile     string
		SendRetries int
	}
	Media struct {
		_mediator.Config `yaml:",inline"`
		LogFile          string
	}
	Aconvert *struct {
		_aconvert.Config `yaml:",inline"`
		LogFile          string
	}
	Reddit *struct {
		reddit.Config `yaml:",inline"`
		LogFile       string
	}
	Dvach *struct {
		Usercode string
		LogFile  string
	}
	Prometheus struct {
		ListenAddress string
	}
}

func main() {
	config := new(Config)
	if err := flu.Read(flu.File(os.Args[1]), util.YAML(config)); err != nil {
		panic(err)
	}

	timeout, err := time.ParseDuration(config.Aggregator.Timeout)
	if err != nil {
		panic(err)
	}

	metrics := _metrics.NewPrometheus(config.Prometheus.ListenAddress).WithPrefix("hikkabot")
	bot := newTelegramBot(config)

	mediator := newMediator(config, metrics.WithPrefix("mediator"))
	defer mediator.Shutdown()

	storage := _storage.NewSQL(config.Aggregator.Storage)
	defer storage.Close()

	channel := feed.Telegram{
		Client: bot.Client,
	}

	agg := &feed.Aggregator{
		Channel:  channel,
		Storage:  storage,
		Mediator: mediator,
		Metrics:  metrics.WithPrefix("aggregator"),
		Timeout:  timeout,
		Aliases:  config.Aggregator.Aliases,
		AdminID:  config.Aggregator.AdminID,
	}

	if config.Dvach != nil {
		client := dvach.NewClient(flu.NewTransport().NewClient(), config.Dvach.Usercode)
		agg.AddSource(_source.DvachCatalog{client, mediator}).
			AddSource(_source.DvachThread{client, mediator})
	}

	if config.Reddit != nil {
		client := reddit.NewClient(flu.NewTransport().NewClient(), config.Reddit.Config)
		source := _source.Reddit{
			Client:   client,
			Mediator: mediator,
			Storage:  storage,
			Metrics:  metrics.WithPrefix("reddit"),
		}
		agg.AddSource(source)
	}

	bot.Listen(config.Telegram.Concurrency, agg.Init().CommandListener(config.Telegram.Username))
}

func newTelegramBot(config *Config) telegram.Bot {
	telegram.SendDelays[telegram.PrivateChat] = time.Second
	telegram.MaxSendRetries = config.Telegram.SendRetries
	bot := telegram.NewBot(flu.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		ProxyURL(config.Telegram.Proxy).
		NewClient().
		Timeout(2*time.Minute), config.Telegram.Token)
	_, err := bot.Send(config.Aggregator.AdminID, &telegram.Text{Text: "⬆️"}, nil)
	if err != nil {
		panic(errors.Wrap(err, "failed to send initial message"))
	}
	return bot
}

func newMediator(config *Config, metrics _metrics.Metrics) *_mediator.Mediator {
	_mediator.CommonClient = flu.NewTransport().
		NewClient().
		AcceptResponseCodes(http.StatusOK).
		Timeout(2 * time.Minute)
	mediator := _mediator.New(config.Media.Config, metrics)
	if config.Aconvert != nil {
		aconvert := _aconvert.NewClient(flu.NewTransport().
			NewClient(), config.Aconvert.Config)
		mediator.AddConverter(_mediator.NewAconverter(aconvert))
	}
	return mediator
}
