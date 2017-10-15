package main

import (
	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/event"
	"log"
)

var levels = map[string]event.Level{
	"debug":	 event.Debug,
	"dbg":		 event.Dbg,
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

func SetUpLogging(config Config) {
	var level event.Level
	if lvl, ok := levels[config.LogLevel]; ok {
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