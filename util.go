package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/event"
)

var unit struct{}

type Config struct {
	Token      string `json:"token"`
	DBFilename string `json:"db_filename"`
	LogLevel   string `json:"log_level"`
}

func GetConfig() (*Config, error) {
	filename := flag.String("config", "", "Configuration file")
	flag.Parse()

	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		return nil, err
	}

	cfg := new(Config)
	if err = json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

var logLevels = map[string]event.Level{
	"debug":     event.Debug,
	"dbg":       event.Dbg,
	"info":      event.Info,
	"notice":    event.Notice,
	"warning":   event.Warning,
	"warn":      event.Warn,
	"error":     event.Error,
	"err":       event.Err,
	"critical":  event.Critical,
	"crit":      event.Crit,
	"alert":     event.Alert,
	"alrt":      event.Alrt,
	"emergency": event.Emergency,
	"emerg":     event.Emerg,
}

// InitLogging configures logging framework
func InitLogging(config *Config) {
	var level event.Level
	if lvl, ok := logLevels[config.LogLevel]; ok {
		level = lvl
	} else {
		level = event.Info
	}

	log.SetOutput(sawmill.NewWriter(level))
	log.SetFlags(0)

	std := sawmill.GetHandler("stdStreams")
	std = sawmill.FilterHandler(std).LevelMin(level)
	sawmill.AddHandler("stdStreams", std)
}

// SignalHandler handles SIGTERM and SIGINT signals
func SignalHandler() *sync.WaitGroup {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		<-signals
		sawmill.Debug("received exit signal")
		wg.Done()
	}()

	return wg
}
