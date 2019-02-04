package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/lego"
	"github.com/jfk9w-go/lego/json"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/service"
	dvach0 "github.com/jfk9w/hikkabot/service/dvach"
	reddit0 "github.com/jfk9w/hikkabot/service/reddit"
	"github.com/jfk9w/hikkabot/service/storage"
	_ "github.com/lib/pq"
)

func main() {
	var config = readConfig()
	initLog(config)

	var (
		bot            = telegram.NewBot(nil, config.Telegram.Token)
		storage        = initStorage(config)
		aggregator     = service.NewAggregator(bot, storage, config.Service.UpdateInterval.Value(), config.Service.Aliases)
		aconvertClient = aconvert.NewClient(nil, &config.Aconvert)
		mediaService   = service.NewMediaService(config.Service.TmpDir, aconvertClient)
		dvachClient    = dvach.NewClient(nil, config.Dvach.Usercode)
		dvachService   = dvach0.NewService(aggregator, mediaService, dvachClient)
		redditClient   = reddit.NewClient(nil, &config.Reddit)
		redditService  = reddit0.Reddit(aggregator, mediaService, redditClient)
	)

	me, err := bot.GetMe()
	lego.Check(err)
	log.Printf("Running as %s", me.Username)

	go func() {
		for _, adminID := range config.Service.AdminIDs {
			_, err = bot.Send(adminID, "⬆️️", telegram.NewSendOpts().Message())
			if err != nil {
				log.Printf("Failed to notify %s about startup: %s", adminID, err)
			}
		}
	}()

	aggregator.
		Add(dvachService.Catalog(), dvachService.Thread(), redditService).
		Init()

	log.Printf("Hikkabot started")

	exit := make(chan os.Signal)
	go signal.Notify(exit, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM)

	go bot.Listen(telegram.NewCommandUpdateListener(bot).
		AddFunc("/status", func(c *telegram.Command) { c.TextReply("I'm alive.") }).
		AddFunc("/sub", aggregator.SubscribeCommandListener).
		AddFunc("/suspend", aggregator.SuspendCommandListener).
		AddFunc("/resume", aggregator.ResumeCommandListener))

	<-exit
	log.Printf("Hikkabot exited")
}

type Storage interface {
	service.Storage
	service.MessageStorage
}

func initStorage(config *Config) Storage {
	c := config.Service.Storage
	if c.Type != "postgres" {
		return storage.Dummy()
	}

	return storage.SQL("postgres", c.DataSource)
}

func initLog(config *Config) {
	if config.Log.Writer != nil {
		path := config.Log.Writer.Value()
		dir := filepath.Dir(path)
		lego.Check(os.MkdirAll(dir, os.ModePerm))
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		lego.Check(err)
		log.SetOutput(file)
	}

	if config.Log.Flags != nil {
		var flags = 0
		for _, key := range *config.Log.Flags {
			flags |= logFlags[key]
		}

		log.SetFlags(flags)
	}
}

func readConfig() *Config {
	if len(os.Args) < 2 {
		panic("no config path specified")
	}

	path := os.ExpandEnv(os.Args[1])
	println("Configuration file path:", path)

	config := new(Config)
	lego.Check(lego.ReadJSONFromFile(path, config))

	return config
}

var logFlags = map[string]int{
	"Ldate":         log.Ldate,
	"Ltime":         log.Ltime,
	"Lmicroseconds": log.Lmicroseconds,
	"Llongfile":     log.Llongfile,
	"Lshortfile":    log.Lshortfile,
	"LUTC":          log.LUTC,
}

type Config struct {
	Log struct {
		Writer *json.Path `json:"writer"`
		Flags  *[]string  `json:"flags"`
	} `json:"log"`

	Service struct {
		UpdateInterval json.Duration `json:"update_interval"`
		TmpDir         string        `json:"tmp"`
		Storage        struct {
			Type       string `json:"type"`
			DataSource string `json:"datasource"`
		} `json:"storage"`
		Aliases  map[telegram.Username]telegram.ID `json:"aliases"`
		AdminIDs []telegram.ID                     `json:"admin_ids"`
	} `json:"service"`

	Telegram struct {
		Token string `json:"token"`
	} `json:"telegram"`

	Dvach struct {
		Usercode string `json:"usercode"`
	} `json:"dvach"`

	Reddit reddit.Config `json:"reddit"`

	Aconvert aconvert.Config `json:"aconvert"`
}
