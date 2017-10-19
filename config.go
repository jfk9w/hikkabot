package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
)

type Config struct {
	Token      string	`json:"token"`
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