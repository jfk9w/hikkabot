package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/lego"
	"github.com/jfk9w-go/lego/json"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/service"
	dvachService "github.com/jfk9w/hikkabot/service/dvach"
	redditService "github.com/jfk9w/hikkabot/service/reddit"
	"github.com/jfk9w/hikkabot/service/storage"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var _ = sqlite3.Version

func main() {
	var config = readConfig()
	initLog(config)

	var (
		bot                 = telegram.NewBot(nil, config.Telegram.Token)
		storage             = initStorage(config)
		aggregator          = service.NewAggregator(storage, bot, config.Service.UpdateInterval.Value())
		fs                  = service.FileSystem(config.Service.TmpDir)
		dvachClient         = dvach.NewClient(nil, config.Dvach.Usercode)
		dvachCatalogService = dvachService.Catalog(aggregator, fs, dvachClient)
		aconvertClient      = aconvert.NewClient(nil, &config.Aconvert)
		dvachThreadService  = dvachService.Thread(aggregator, fs, storage, dvachClient, aconvertClient)
		redditClient        = reddit.NewClient(nil, &config.Reddit)
		redditService       = redditService.Reddit(aggregator, fs, redditClient)
	)

	aggregator.
		Add(dvachCatalogService, dvachThreadService, redditService).
		Init()

	log.Printf("Hikkabot started")

	var exit sync.WaitGroup
	exit.Add(1)
	go func() {
		defer exit.Done()
		var ch = make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM)
		<-ch
	}()

	go bot.Listen(telegram.NewCommandUpdateListener(bot).
		AddFunc("/status", func(c *telegram.Command) { c.TextReply("I'm alive.") }).
		AddFunc("/sub", aggregator.SubscribeCommandListener).
		AddFunc("/suspend", aggregator.SuspendCommandListener).
		AddFunc("/resume", aggregator.ResumeCommandListener))

	exit.Wait()
	log.Printf("Hikkabot exited")
}

type Storage interface {
	service.Storage
	dvachService.PostMessageRefStorage
}

func initStorage(config *Config) Storage {
	c := config.Service.Storage
	if c.Type != "sqlite3" {
		return storage.Dummy()
	}

	return storage.SQL("sqlite3", c.DataSource)
}

func initLog(config *Config) {
	if config.Log.Writer != nil {
		path := config.Log.Writer.Value()
		dir := filepath.Dir(path)
		lego.Check(os.MkdirAll(dir, os.ModePerm))
		file, err := os.Create(path)
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
