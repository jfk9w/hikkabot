package main

import (
	"flag"
	sm "github.com/phemmer/sawmill"
	"net/http"
	"time"
)

var HttpClient = new(http.Client)

type Config struct {
	Token    string
	Snapshot string
	LogLevel string
}

func GetConfig() Config {
	token := flag.String("token", "", "Telegram Bot API token")
	snapshot := flag.String("snapshot", "", "Snapshot file location")
	logLevel := flag.String("log", "info", "Set the log level")
	flag.Parse()

	return Config{
		Token: *token,
		Snapshot: *snapshot,
		LogLevel: *logLevel,
	}
}

func main() {
	defer func() {
		sm.CheckPanic()
		sm.Stop()
	}()

	cfg := GetConfig()
	SetUpLogging(cfg)

	ctl := SetUp(cfg)
	ctl.Start()

	time.Sleep(time.Minute)

	ctl.Stop()
}