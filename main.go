package main

import (
	"flag"
	"net/http"
	"time"

	"encoding/json"
	"github.com/phemmer/sawmill"
	"io/ioutil"
)

var HttpClient *http.Client = new(http.Client)

type Config struct {
	Token      string	`json:"telegram_bot_api_token"`
	DBFilename string	`json:"db_filename"`
	LogLevel   string	`json:"log_level"`
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

func main() {
	defer func() {
		sawmill.CheckPanic()
		sawmill.Stop()
	}()

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	InitLogging(cfg)

	ctl := InitController(cfg)
	time.Sleep(time.Minute)
	ctl.Stop()
}
