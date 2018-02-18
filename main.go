package main

import (
	"net/http"
	"os"
	"time"

	"github.com/dgraph-io/badger"
	dv "github.com/jfk9w/hikkabot/dvach"
	"github.com/jfk9w/hikkabot/service"
	"github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/telegram"
	"github.com/jfk9w/hikkabot/webm"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Token    string `json:"token"`
	DB       string `json:"db_filename"`
	LogLevel string `json:"log_level"`
}

func main() {
	InitLogging(cfg)

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

	service.Init(bot, dvach, cfg.DB)

	conv, hConv = webm.Converter(webm.Wrap(httpc))
	hCtl := Controller(bot)

	SignalHandler().Wait()
	hCtl.Ping()
	bot.Stop()
	client.Stop()
}
