package telegram

import (
	"errors"
	"sync"
	"time"
)

const (
	UpdatesAlreadyRunning = "updates already running"
	UpdatesNotRunning = "updates not running"
)

type Updates struct {
	C chan Update

	gateway *Gateway
	request GetUpdatesRequest
	wg      *sync.WaitGroup
	mu      *sync.Mutex
	stop    chan struct{}
}

func NewUpdates(gateway *Gateway, base GetUpdatesRequest) *Updates {
	return &Updates{
		C:       make(chan Update, 20),
		gateway: gateway,
		request: base,
		mu:      new(sync.Mutex),
		stop:    make(chan struct{}, 1),
	}
}

func (u *Updates) Start() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.wg != nil {
		return errors.New(UpdatesAlreadyRunning)
	}

	u.wg = new(sync.WaitGroup)
	u.wg.Add(1)
	go func() {
		ticker := time.NewTicker(u.request.timeout)
		defer func() {
			ticker.Stop()
			u.wg.Done()
		}()

		for {
			select {
			case <-u.stop:
				return

			case <-ticker.C:
				resp, err := u.gateway.MakeRequest(u.request)
				if err != nil || !resp.Ok {
					// logging
					continue
				}

				updates := make([]Update, 0)
				err = resp.Parse(updates)
				if err != nil {
					// logging
					continue
				}

				for _, update := range updates {
					u.C <- update
				}
			}
		}
	}()

	return nil
}

func (u *Updates) Stop() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.wg == nil {
		return errors.New(UpdatesNotRunning)
	}

	u.stop <- unit
	u.wg.Wait()
	u.wg = nil

	return nil
}