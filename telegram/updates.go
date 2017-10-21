package telegram

import (
	"time"
	"github.com/phemmer/sawmill"
)

type updates struct {
	c       chan Update
	gateway *gateway
	request GetUpdatesRequest
	stop0   chan struct{}
	done    chan struct{}
}

func (svc *updates) Channel() <-chan Update {
	return svc.c
}

func newUpdates(gateway *gateway, base GetUpdatesRequest) *updates {
	return &updates{
		c:       make(chan Update, 20),
		gateway: gateway,
		request: base,
		stop0:   make(chan struct{}, 1),
		done:    make(chan struct{}, 1),
	}
}

func (svc *updates) start() {
	go func() {
		defer func() {
			sawmill.Info("telegram.updates.stop")
			svc.done <- unit
		}()

		for {
			select {
			case <-svc.stop0:
				return

			default:
				resp, err := svc.gateway.makeRequest(svc.request)
				if err != nil || !resp.Ok {
					time.Sleep(3 * time.Second)
					continue
				}

				updates := make([]Update, 0)
				err = resp.Parse(&updates)
				if err != nil {
					// logging
					continue
				}

				for _, update := range updates {
					svc.c <- update
					offset := update.ID + 1
					if svc.request.Offset < offset {
						svc.request.Offset = offset
					}
				}
			}
		}
	}()

	sawmill.Info("telegram.updates.start")
}

func (svc *updates) stop() <-chan struct{} {
	svc.stop0 <- unit
	return svc.done
}
