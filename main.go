package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"

	"github.com/jfk9w-go/hikkabot/common/gox"

	Aconvert "github.com/jfk9w-go/hikkabot/common/aconvert-api"
	Dvach "github.com/jfk9w-go/hikkabot/common/dvach-api"
	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	"github.com/jfk9w-go/hikkabot/common/logx"
	Red "github.com/jfk9w-go/hikkabot/common/reddit-api"
	Telegram "github.com/jfk9w-go/hikkabot/common/telegram-bot-api"
	Engine "github.com/jfk9w-go/hikkabot/engine"
	"github.com/jfk9w-go/hikkabot/frontend"
)

func main() {
	if len(os.Args) < 2 {
		panic("config path is not specified")
	}

	var (
		config = ReadConfig(os.Args[1])

		aconvert = Aconvert.ConfigureBalancer(config.Aconvert)
		dvach    = Dvach.Configure(config.Dvach)
		red      = Red.Configure(config.Red.Config)
		telegram = Telegram.Configure(config.Telegram, &Telegram.UpdatesOpts{
			Timeout:        60,
			AllowedUpdates: []string{"message", "edited_message"},
		})

		context = &Engine.Context{telegram, dvach, &aconvert, red}
	)

	var redMetricsFile = config.Red.MetricsFile
	if redMetricsFile != "" {
		var err error
		redMetricsFile, err = fsx.Path(redMetricsFile)
		gox.Check(err)
		gox.Check(fsx.EnsureParent(redMetricsFile))
	}

	var engine = Engine.New(context, config.SchedulerInterval.Duration(), config.Database,
		redMetricsFile, config.Red.MetricsChatID)

	frontend.Init(engine, context, config.Frontend)

	logx.Get("init").Debug("Started")

	go profiler()
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

func profiler() {
	runtime.SetBlockProfileRate(10)
	runtime.SetMutexProfileFraction(10)
	logx.Get("profiler").Println(http.ListenAndServe("0.0.0.0:6060", nil))
}
