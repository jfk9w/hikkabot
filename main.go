package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/backend"
	"github.com/jfk9w-go/hikkabot/bot"
	"github.com/jfk9w-go/hikkabot/frontend"
	"github.com/jfk9w-go/hikkabot/keeper"
	"github.com/jfk9w-go/logrus"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
)

var log = logrus.GetLogger("main")

type Config struct {
	BackendGCTimeout    int `json:"backend_gc_timeout"`
	AconvertReadTimeout int `json:"aconvert_read_timeout"`

	Keeper   keeper.Config   `json:"keeper"`
	Telegram telegram.Config `json:"telegram"`
	Dvach    dvach.Config    `json:"dvach"`
	Aconvert aconvert.Config `json:"aconvert"`
}

func readConfig() *Config {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("CONFIG not set")
	}

	cfg := new(Config)
	if err := misc.ReadJSON(path, cfg); err != nil {
		panic(err)
	}

	return cfg
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}

		log.Infof("Exit")
	}()

	// Config
	cfg := readConfig()

	// Keeper
	db := keeper.NewKeeper()
	fsync, err := keeper.RunFileSync(db, cfg.Keeper)
	if err != nil {
		panic(err)
	}

	defer func() {
		fsync.Close()
		fsync.Save()
	}()

	// Frontend
	bot0 := telegram.Configure(cfg.Telegram)
	conv := aconvert.Configure(cfg.Aconvert)
	botx := bot.Wrap(bot0, conv, millis(cfg.AconvertReadTimeout))
	dvch := dvach.Configure(cfg.Dvach)
	ff := backend.NewFeedFactory(botx, dvch, conv, db)

	back := backend.Run(botx, ff)
	front := frontend.New(botx, dvch, back)
	for chat, threads := range db.GetOffsets() {
		for thread, offset := range threads {
			hash, err := front.Hashtag(thread)
			if err != nil {
				db.DeleteOffset(chat, thread)
				log.Warningf("Unable to re-subscribe to %s: %s", thread, err)
				continue
			}

			back.Subscribe(chat, thread, hash, offset)
		}
	}

	go back.GC(millis(cfg.BackendGCTimeout))
	go front.Run()

	// Signal handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	misc.BroadcastCloser(conv, bot0)
}

func millis(value int) time.Duration {
	return time.Duration(value) * time.Millisecond
}
