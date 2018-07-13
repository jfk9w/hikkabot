package main

import (
	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox/fsx"
	"github.com/jfk9w-go/gox/jsonx"
	"github.com/jfk9w-go/logx"
	"github.com/jfk9w-go/telegram"
)

type Config struct {
	Database          string            `json:"database"`
	Superusers        []telegram.ChatID `json:"superusers"`
	SchedulerInterval jsonx.Duration    `json:"scheduler_interval"`
	Dvach             dvach.Config      `json:"dvach"`
	Telegram          telegram.Config   `json:"telegram"`
	Aconvert          aconvert.Config   `json:"aconvert"`
}

func ReadConfig(path string) *Config {
	var err error
	path, err = fsx.Path(path)
	checkpanic(err)

	logx.Get("init").Debugf("Reading config from %s", path)

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
