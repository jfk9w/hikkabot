package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/phemmer/sawmill"
)

var HttpClient = new(http.Client)

type Config struct {
	Token      string
	DbFilename string
	LogLevel   string
}

func GetConfig() Config {
	token := flag.String("token", "", "Telegram Bot API token")
	dbfilename := flag.String("db", "", "Database file location")
	logLevel := flag.String("log", "info", "Set the log level")
	flag.Parse()

	return Config{
		Token:      *token,
		DbFilename: *dbfilename,
		LogLevel:   *logLevel,
	}
}

func main() {
	defer func() {
		sawmill.CheckPanic()
		sawmill.Stop()
	}()

	cfg := GetConfig()
	SetUpLogging(cfg)

	ctl := SetUp(cfg)
	ctl.Start()

	time.Sleep(time.Minute)

	ctl.Stop()
}
