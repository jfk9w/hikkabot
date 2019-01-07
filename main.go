package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/hikkabot/api/dvach"
	"github.com/jfk9w-go/hikkabot/api/reddit"
	. "github.com/jfk9w-go/hikkabot/service"
	"github.com/jfk9w-go/lego"
	"github.com/jfk9w-go/lego/json"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var _ = sqlite3.Version

func main() {
	var config = readConfig()
	initLog(config)

	var (
		bot                 = telegram.NewBot(nil, config.Telegram.Token)
		storage             = NewSqlStorage("sqlite3", config.DataSource)
		scheduler           = NewScheduler(storage, bot, config.SchedulerInterval.Value())
		baseSubscribe       = BaseSubscribe(storage, scheduler, bot)
		fileSystem          = FileSystem(config.TempDir)
		dvachClient         = dvach.NewClient(nil, config.Dvach.Usercode)
		dvachCatalogService = DvachCatalog(baseSubscribe, dvachClient)
		aconvertClient      = aconvert.NewClient(nil, &config.Aconvert)
		redditClient        = reddit.NewClient(nil, &config.Reddit)
		redditService       = Reddit(baseSubscribe, fileSystem, redditClient)
	)

	scheduler.Register(dvachCatalogService, redditService).Init()

	log.Printf("Bonobo started")

	var exit sync.WaitGroup
	exit.Add(1)
	go func() {
		defer exit.Done()
		var ch = make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGABRT, syscall.SIGTERM)
		<-ch
	}()

	go bot.Listen(telegram.NewCommandUpdateListener().
		AddFunc("/status", func(c *telegram.Command) {
			c.TextReply("I'm alive.")
		}).
		Add("/subscribe", new(SubscribeCommandListener).
			Add(dvachCatalogService, redditService)))

	exit.Wait()

	_ = aconvertClient

	log.Printf("Bonobo exited")
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

	TempDir           string        `json:"temp_dir"`
	DataSource        string        `json:"datasource"`
	SchedulerInterval json.Duration `json:"scheduler_interval"`

	Telegram struct {
		Token string `json:"token"`
	} `json:"telegram"`

	Dvach struct {
		Usercode string `json:"usercode"`
	} `json:"dvach"`

	Reddit reddit.Config `json:"reddit"`

	Aconvert aconvert.Config `json:"aconvert"`
}
