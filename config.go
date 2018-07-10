package main

import (
	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/fsx"
	"github.com/jfk9w-go/gox/jsonx"
	"github.com/jfk9w-go/telegram"
)

type Config struct {
	Database          string          `json:"database"`
	SchedulerInterval jsonx.Duration  `json:"scheduler_interval"`
	Dvach             dvach.Config    `json:"2ch.hk"`
	Telegram          telegram.Config `json:"telegram"`
	Aconvert          aconvert.Config `json:"aconvert.com"`
}

func ReadConfig(path string) *Config {
	var err error
	path, err = fsx.Path(path)
	checkpanic(err)

	println("Reading config from " + path)

	var config = new(Config)
	checkpanic(jsonx.ReadFile(path, config))

	if err != nil {
		panic(err)
	}

	return config
}

func checkpanic(err error) {
	if err != nil {
		panic(err)
	}
}
