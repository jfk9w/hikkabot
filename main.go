package main

import (
	_ "net/http/pprof"
	"os"
	"time"

	_aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	_media "github.com/jfk9w/hikkabot/media"
	_metrics "github.com/jfk9w/hikkabot/metrics"
	_source "github.com/jfk9w/hikkabot/source"
	_storage "github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/util"
	"github.com/pkg/errors"
)

type Size struct {
	Bytes     int64
	Kilobytes int64
	Megabytes int64
}

func (s *Size) Value(defaultValue int64) int64 {
	if s == nil {
		return defaultValue
	} else {
		return s.Megabytes<<20 + s.Kilobytes<<10 + s.Bytes
	}
}

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
		Concurrency      int
		MinSize, MaxSize Size
		Buffer           bool
		Directory        string
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

	storage := _storage.NewSQL(config.Aggregator.Storage)
	defer storage.Close()

	mediator := &_media.Tor{
		Metrics:     metrics.WithPrefix("mediator"),
		Storage:     storage,
		BufferSpace: _media.BufferSpace(config.Media.Directory),
		SizeBounds: [2]int64{
			config.Media.MinSize.Value(1 << 10),
			config.Media.MaxSize.Value(75 << 20),
		},
		Buffer:  config.Media.Buffer,
		Debug:   true,
		Workers: config.Media.Concurrency,
	}

	if config.Aconvert != nil {
		mediator.AddConverter(_media.NewAconvertConverter(config.Aconvert.Config, config.Media.Directory))
	}

	defer mediator.Initialize().Close()

	channel := feed.Telegram{
		Client: bot.Client,
	}

	agg := &feed.Aggregator{
		Channel: channel,
		Storage: storage,
		Tor:     mediator,
		Metrics: metrics.WithPrefix("aggregator"),
		Timeout: timeout,
		Aliases: config.Aggregator.Aliases,
		AdminID: config.Aggregator.AdminID,
	}

	if config.Dvach != nil {
		client := dvach.NewClient(flu.NewTransport().NewClient(), config.Dvach.Usercode)
		agg.AddSource(_source.DvachCatalog{client, mediator}).
			AddSource(_source.DvachThread{client, mediator})
	}

	if config.Reddit != nil {
		client := reddit.NewClient(flu.NewTransport().NewClient(), config.Reddit.Config)
		source := _source.Reddit{
			Client:  client,
			Tor:     mediator,
			Storage: storage,
			Metrics: metrics.WithPrefix("reddit"),
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
