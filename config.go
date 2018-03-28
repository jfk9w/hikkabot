package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
)

type Config struct {
	Tokens   []string `json:"tokens"`
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
