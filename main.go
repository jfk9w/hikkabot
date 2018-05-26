package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"path/filepath"

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

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}

		log.Infof("Exit")
	}()

	// Config
	token := os.Getenv("TOKEN")
	host := os.Getenv("HOST")
	root := os.Getenv("ROOT")
	domain := os.Getenv("DOMAIN")
	hiddenBoards := os.Getenv("HIDDEN_BOARDS")

	// Keeper
	path := os.Getenv("KEEPER")
	db := keeper.NewKeeper()
	fsync, err := keeper.RunFileSync(db, 30*time.Second, path)
	if err != nil {
		panic(err)
	}

	defer func() {
		fsync.Close()
		fsync.Save()
	}()

	// Frontend
	bot0 := telegram.New(telegram.DefaultConfig.WithToken(token))
	conv := aconvert.WithCache(3*24*time.Hour, 1*time.Minute, 12*time.Hour)
	botx := bot.Wrap(bot0, conv)
	dvch := dvach.New(dvach.NewProxy(host, expand(root), domain).Run(), strings.Split(hiddenBoards, ",")...)
	ff := backend.NewFeedFactory(botx, dvch, conv, db)

	back := backend.Run(botx, ff)
	front := frontend.New(botx, dvch, back)
	for chat, threads := range db.GetOffsets() {
		for thread, offset := range threads {
			hash, err := front.Hashtag(thread)
			if err != nil {
				log.Warningf("Unable to re-subscribe to %s: %s", thread, err)
				continue
			}

			back.Subscribe(chat, thread, hash, offset)
		}
	}

	go back.GC(5 * time.Minute)
	go front.Run()

	// Signal handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	misc.BroadcastCloser(conv, bot0)
}

func expand(path string) string {
	path, _ = filepath.Abs(os.ExpandEnv(path))
	return path
}
