package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/jfk9w/hikkabot/controller"
	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	tg "github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/util"
	"github.com/jfk9w/hikkabot/webm"
	log "github.com/sirupsen/logrus"
)

func main() {
	defer log.Info("MAIN exit")

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	log.SetOutput(os.Stdout)
	lvl, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		panic(err)
	}

	log.SetLevel(lvl)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true})

	opts := badger.DefaultOptions
	opts.Dir = cfg.DB
	opts.ValueDir = cfg.DB
	db, err := service.NewBadgerStorage(
		service.Config{
			ThreadTTL: 3 * 24 * time.Hour,
			WebmTTL:   3 * 24 * time.Hour,
		},
		opts,
	)

	if err != nil {
		panic(err)
	}

	state, err := db.Load()
	if err != nil {
		panic(err)
	}

	defer db.Close()

	httpc := new(http.Client)
	dvach := dv.New(httpc)
	bot, err := tg.New(
		httpc,
		cfg.Token,
		tg.GetUpdatesRequest{
			Timeout:        60,
			AllowedUpdates: []string{"message"},
		},
	)

	if err != nil {
		panic(err)
	}

	defer bot.Stop()

	conv, hConv := webm.Converter(webm.Wrap(httpc), db, 7, 6)
	defer hConv.Ping()

	svc := service.New(dvach, bot, conv, db)
	svc.Init(state)
	defer svc.Stop()

	hCtl := controller.Start(bot, svc)
	defer hCtl.Ping()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	exit := make(chan util.UnitType, 1)
	go func(c chan<- util.UnitType) {
		<-signals
		c <- util.Unit
	}(exit)

	<-exit
}
