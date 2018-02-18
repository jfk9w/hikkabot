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
	Token    string `json:"token"`
	DB       string `json:"db"`
	LogLevel string `json:"log_level"`
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
