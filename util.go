package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

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

// InitLogging configures logging framework
func InitLogging(config *Config) {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.ParseLevel(config.LogLevel))
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

// SignalHandler handles SIGTERM and SIGINT signals
func SignalHandler() util.Hook {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	hook := util.NewHook()
	go func() {
		<-signals
		hook.Send()
		sawmill.Debug("received exit signal")
	}()

	return hook
}
