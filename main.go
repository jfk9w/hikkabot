package main

import (
	"os"

	"os/signal"
	"syscall"

	"github.com/jfk9w-go/hikkabot/frontend"
	"github.com/jfk9w-go/logrus"
)

var log = logrus.GetLogger("main")

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Panic(err)
		}

		log.Infof("Exit")
	}()

	// Config
	token := os.Getenv("TOKEN")
	//chat := telegram.NewChatRef(os.Getenv("CHAT"))

	// Frontend
	front := frontend.Run(token)

	// Signal handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	front.Close()
}
