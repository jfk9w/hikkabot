package main

import (
	"github.com/jfk9w-go/hikkabot/common/aconvert-api"
	"github.com/jfk9w-go/hikkabot/common/dvach-api"
	"github.com/jfk9w-go/hikkabot/common/gox"
	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	"github.com/jfk9w-go/hikkabot/common/gox/jsonx"
	"github.com/jfk9w-go/hikkabot/common/logx"
	"github.com/jfk9w-go/hikkabot/common/reddit-api"
	"github.com/jfk9w-go/hikkabot/common/telegram-bot-api"
	"github.com/jfk9w-go/hikkabot/frontend"
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
