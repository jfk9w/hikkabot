package telegram

import (
	"time"

	"github.com/jfk9w/hikkabot/util"
	"github.com/phemmer/sawmill"
)

type updates struct {
	c       chan Update
	gateway *gateway
	request GetUpdatesRequest

	halt util.Hook
	done util.Hook
}

func (svc *updates) Channel() <-chan Update {
	return svc.c
}

func newUpdates(gateway *gateway, base GetUpdatesRequest) *updates {
	return &updates{
		c:       make(chan Update, 20),
		gateway: gateway,
		request: base,
		halt:    util.NewHook(),
		done:    util.NewHook(),
	}
}

func (svc *updates) start() {
	go func() {
		defer func() {
			svc.done.Send()
		}()

		for {
			select {
			case <-svc.halt:
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

	sawmill.Notice("updates started")
}

func (svc *updates) stop() {
	svc.halt.Send()
	svc.done.Wait()

	sawmill.Notice("updates stopped")
}
