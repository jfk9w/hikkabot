package main

import (
	"github.com/phemmer/sawmill"
)

func main() {
	defer func() {
		sawmill.CheckPanic()
		sawmill.Stop()
	}()

	cfg, err := GetConfig()
	if err != nil {
		panic(err)
	}

	InitLogging(cfg)

	ctl := InitController(cfg)
	SignalHandler().Wait()
	<-ctl.Stop()
}