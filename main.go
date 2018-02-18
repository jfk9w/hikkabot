package main

import (
	"net/http"
	"os"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/jfk9w/hikkabot/controller"
	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/webm"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.ParseLevel(config.LogLevel))
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	badgerOpts := badger.DefaultOptions
	badgerOpts.Dir = cfg.DB
	badgerDB := badger.Open(badgerOpts)
	defer badgerDB.Close()

	db := storage.New(
		storage.Config{
			InactiveTTL: 3 * 24 * time.Hour,
			VideoTTL:    3 * 24 * time.Hour,
		},
		badgerDB,
	)

	httpc := new(http.Client)
	dvach := dv.New(httpc)
	bot, err := telegram.New(
		httpc,
		cfg.Token,
		telegram.GetUpdatesRequest{
			Timeout:        60,
			AllowedUpdates: []string{"message"},
		},
	)

	if err != nil {
		panic(err)
	}

	defer bot.Stop()

	conv, hConv = webm.Converter(webm.Wrap(httpc), 7, 6)
	defer hConv.Ping()

	hCtl := controller.Start(bot)
	defer hCtl.Pind()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	exit := make(chan util.UnitType, 1)
	go func(c chan<- util.UnitType) {
		<-signals
		c <- Unit
	}(exit)

	<-exit
	log.Debug("MAIN exit")
}
