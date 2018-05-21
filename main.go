package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/backend"
	"github.com/jfk9w-go/hikkabot/bot"
	"github.com/jfk9w-go/hikkabot/frontend"
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
	cert := os.Getenv("CERT")
	key := os.Getenv("KEY")
	hiddenBoards := os.Getenv("HIDDEN_BOARDS")

	// Frontend
	bot0 := telegram.New(telegram.DefaultConfig.WithToken(token))
	conv := aconvert.WithCache(3*24*time.Hour, 1*time.Minute, 12*time.Hour)
	botx := bot.Wrap(bot0, conv)
	dvch := dvach.New(dvach.NewProxy(host, root, domain, cert, key).Run(), strings.Split(hiddenBoards, ",")...)
	ff := backend.NewFeedFactory(botx, dvch, conv)
	back := backend.Run(botx, ff)
	frontend.Run(botx, dvch, back)

	// Signal handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	misc.BroadcastCloser(conv, bot0)
}
