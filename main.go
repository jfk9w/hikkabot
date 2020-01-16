package main

import (
	"expvar"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	_aconvert "github.com/jfk9w-go/aconvert-api"
	"github.com/jfk9w-go/flu"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w/hikkabot/api/dvach"
	"github.com/jfk9w/hikkabot/api/reddit"
	"github.com/jfk9w/hikkabot/feed"
	_mediator "github.com/jfk9w/hikkabot/mediator"
	"github.com/jfk9w/hikkabot/source"
	_storage "github.com/jfk9w/hikkabot/storage"
	"github.com/jfk9w/hikkabot/util"
	"github.com/pkg/errors"
)

type Config struct {
	Aggregator struct {
		AdminID telegram.ID
		Aliases map[telegram.Username]telegram.ID
		Storage _storage.SQLConfig
		Timeout string
	}
	Telegram struct {
		Username    string
		Token       string
		Proxy       string
		Concurrency int
		LogFile     string
		SendRetries int
	}
	Media struct {
		_mediator.Config `yaml:",inline"`
		LogFile          string
	}
	Aconvert *struct {
		_aconvert.Config `yaml:",inline"`
		LogFile          string
	}
	Reddit *struct {
		reddit.Config `yaml:",inline"`
		LogFile       string
	}
	Dvach *struct {
		Usercode string
		LogFile  string
	}
}

func init() {
	launch := time.Now()
	expvar.NewString("launch").Set(launch.Format(time.RFC3339))
	expvar.Publish("uptime", expvar.Func(func() interface{} { return time.Now().Sub(launch).String() }))
}

func main() {
	config := new(Config)
	err := flu.Read(flu.File(os.Args[1]), util.YAML(config))
	if err != nil {
		panic(err)
	}
	timeout, err := time.ParseDuration(config.Aggregator.Timeout)
	if err != nil {
		panic(err)
	}
	logging := make(Logging)
	defer logging.Close()
	requests := new(sync.Map)
	http.Handle("/debug/requests", DebugHTTPHandler{requests})
	go func() { log.Println(http.ListenAndServe("localhost:6060", nil)) }()
	telegram.SendDelays[telegram.PrivateChat] = time.Second
	telegram.MaxSendRetries = config.Telegram.SendRetries
	bot := telegram.NewBot(flu.NewTransport().
		ResponseHeaderTimeout(2*time.Minute).
		ProxyURL(config.Telegram.Proxy).
		Logger(logging.Get(config.Telegram.LogFile)).
		PendingRequests(requests).
		NewClient().
		Timeout(2*time.Minute), config.Telegram.Token)
	_mediator.CommonClient = flu.NewTransport().
		Logger(logging.Get(config.Media.LogFile)).
		PendingRequests(requests).
		NewClient().
		AcceptResponseCodes(http.StatusOK).
		Timeout(2 * time.Minute)
	mediator := _mediator.New(config.Media.Config)
	defer mediator.Shutdown()
	if config.Aconvert != nil {
		aconvert := _aconvert.NewClient(flu.NewTransport().
			Logger(logging.Get(config.Aconvert.LogFile)).
			PendingRequests(requests).
			NewClient(), config.Aconvert.Config)
		mediator.AddConverter(_mediator.NewAconverter(aconvert))
	}
	storage := _storage.NewSQL(config.Aggregator.Storage)
	defer storage.Close()
	_, err = bot.Send(config.Aggregator.AdminID, &telegram.Text{Text: "⬆️"}, nil)
	if err != nil {
		panic(errors.Wrap(err, "failed to send initial message"))
	}
	agg := &feed.Aggregator{
		Channel:  feed.Telegram{Client: bot.Client},
		Storage:  storage,
		Mediator: mediator,
		Timeout:  timeout,
		Aliases:  config.Aggregator.Aliases,
		AdminID:  config.Aggregator.AdminID,
	}
	if config.Dvach != nil {
		client := dvach.NewClient(flu.NewTransport().
			Logger(logging.Get(config.Dvach.LogFile)).
			PendingRequests(requests).
			NewClient(), config.Dvach.Usercode)
		agg.AddSource(source.DvachCatalog{client}).
			AddSource(source.DvachThread{client})
	}
	if config.Reddit != nil {
		client := reddit.NewClient(flu.NewTransport().
			Logger(logging.Get(config.Reddit.LogFile)).
			PendingRequests(requests).
			NewClient(), config.Reddit.Config)
		agg.AddSource(source.Reddit{client})
	}
	bot.Listen(config.Telegram.Concurrency, agg.Init().CommandListener(config.Telegram.Username))
}

type Logging map[string]*os.File

func (logging Logging) Get(path string) *log.Logger {
	if path == "" {
		return nil
	}
	if path == "stdout" {
		return log.New(log.Writer(), "", log.Flags())
	}
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	if _, ok := logging[path]; !ok {
		err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			panic(err)
		}
		file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0)
		if err != nil {
			panic(err)
		}
		logging[path] = file
	}
	return log.New(logging[path], "", log.Flags())
}

func (logging Logging) Close() {
	for _, file := range logging {
		_ = file.Close()
	}
}

type DebugHTTPHandler struct {
	requests *sync.Map
}

func (h DebugHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		return
	}
	var err error
	switch req.URL.Path {
	case "/debug/requests":
		err = h.handleRequests(rw)
	default:
		return
	}
	if err != nil {
		rw.Header().Add("X-Hikkabot-Error", err.Error())
	} else {
		rw.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	}
}

func (h DebugHTTPHandler) handleRequests(rw http.ResponseWriter) error {
	var err error
	h.requests.Range(func(k, v interface{}) bool {
		if err != nil {
			return false
		}
		_, err = rw.Write([]byte(k.(string) + "\t" + v.(string) + "\n"))
		return true
	})
	return err
}
