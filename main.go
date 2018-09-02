package main

import (
	"os"
	"os/signal"
	"sync"

	Aconvert "github.com/jfk9w-go/aconvert"
	Dvach "github.com/jfk9w-go/dvach"
	Engine "github.com/jfk9w-go/hikkabot/engine"
	"github.com/jfk9w-go/hikkabot/frontend"
	"github.com/jfk9w-go/logx"
	Telegram "github.com/jfk9w-go/telegram"
)

func main() {
	if len(os.Args) < 2 {
		panic("config path is not specified")
	}

	var (
		config = ReadConfig(os.Args[1])

		aconvert = Aconvert.ConfigureBalancer(config.Aconvert)
		dvach    = Dvach.Configure(config.Dvach)
		telegram = Telegram.Configure(config.Telegram, &Telegram.UpdatesOpts{
			Timeout:        60,
			AllowedUpdates: []string{"message", "edited_message"},
		})

		context = &Engine.Context{telegram, dvach, &aconvert}
		engine  = Engine.New(context, config.SchedulerInterval.Duration(), config.Database)
	)

	frontend.Init(engine, config.Superusers)

	logx.Get("init").Debug("Started")

	loop()

	//telegram.Updater.Close()
	//aconvert.Close()
	engine.DB.Close()

	println("Shutdown")
}

func loop() {
	var (
		s     = make(chan os.Signal)
		group sync.WaitGroup
	)

	group.Add(1)
	go func() {
		signal.Notify(s, os.Interrupt, os.Kill)
		<-s
		group.Done()
	}()

	group.Wait()
}
