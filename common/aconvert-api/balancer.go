package aconvert

import (
	"github.com/jfk9w-go/hikkabot/common/gox/closer"
	"github.com/jfk9w-go/hikkabot/common/gox/serialx"
	"github.com/jfk9w-go/hikkabot/common/gox/unit"
)

var servers = []string{"3", "5", "7", "9", "11", "13", "15"}

type Balancer struct {
	*API
	closer.I
	queue chan *serialx.Item
	retry int
}

func ConfigureBalancer(config Config) Balancer {
	var (
		queue = make(chan *serialx.Item)
		api   = NewAPI(config.Http)
	)

	mons := make([]closer.I, len(servers))
	for i, server := range servers {
		mon := unit.NewChan()
		mons[i] = mon
		go func(server string, mon unit.Chan) {
			for {
				select {
				case <-mon.Out():
					return

				case item := <-queue:
					if !item.Resolve(server) {
						queue <- item
						continue
					}
				}
			}
		}(server, mon)
	}

	return Balancer{api, closer.Broadcast(mons...), queue, config.Retries}
}

func (b *Balancer) Convert(input interface{}) (string, error) {
	item := serialx.NewItem(func(any interface{}) serialx.Out {
		server := any.(string)
		url, err := b.API.Convert(server, input)
		return &balancerResponse{url, err}
	}, b.retry)

	b.queue <- item

	resp := item.Out().(*balancerResponse)
	return resp.URL, resp.Error
}
