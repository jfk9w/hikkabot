package main

import (
	"os"

	"github.com/jfk9w-go/aconvert"
	"github.com/jfk9w-go/dvach"
	"github.com/jfk9w-go/hikkabot/keeper"
	"github.com/jfk9w-go/misc"
	"github.com/jfk9w-go/telegram"
)

type Config struct {
	BackendGCTimeout    int `json:"backend_gc_timeout"`
	AconvertReadTimeout int `json:"aconvert_read_timeout"`

	Keeper   keeper.Config   `json:"keeper"`
	Telegram telegram.Config `json:"telegram"`
	Dvach    dvach.Config    `json:"dvach"`
	Aconvert aconvert.Config `json:"aconvert"`
}

func readConfig() *Config {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("CONFIG not set")
	}

	cfg := new(Config)
	if err := misc.ReadJSON(path, cfg); err != nil {
		panic(err)
	}

	return cfg
}
