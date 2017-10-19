package main

import (
	"github.com/phemmer/sawmill"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var unit struct{}

func SignalHandler() *sync.WaitGroup {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		<-signals
		sawmill.Debug("Received SIGTERM")
		wg.Done()
	}()

	return wg
}
