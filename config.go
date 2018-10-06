package main

import (
	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/gox"
	"github.com/jfk9w-go/gox/fsx"
	"github.com/jfk9w-go/gox/jsonx"
	"github.com/jfk9w-go/hikkabot/frontend"
	"github.com/jfk9w-go/logx"
	"github.com/jfk9w-go/red"
	"github.com/jfk9w-go/telegram"
)

type Config struct {
	Database          string          `json:"database"`
	SchedulerInterval jsonx.Duration  `json:"scheduler_interval"`
	Frontend          frontend.Config `json:"frontend"`
	Dvach             dvach.Config    `json:"dvach"`
	Telegram          telegram.Config `json:"telegram"`
	Aconvert          aconvert.Config `json:"aconvert"`
	Red               RedConfig       `json:"red"`
}

type RedConfig struct {
	red.Config
	MetricsFile   string          `json:"metrics_file"`
	MetricsChatID telegram.ChatID `json:"metrics_chat_id"`
}

func ReadConfig(path string) *Config {
	var err error
	path, err = fsx.Path(path)
	gox.Check(err)

	logx.Get("init").Debugf("Reading config from %s", path)

	var config = new(Config)
	gox.Check(jsonx.ReadFile(path, config))

	if err != nil {
		panic(err)
	}

	config.Telegram.RouterConfig = telegram.DefaultIntervals
	return config
}
