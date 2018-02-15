package webm

import (
	"sync"

	"github.com/jfk9w/hikkabot/util"
	log "github.com/sirupsen/logrus"
)

type (
	Loader interface {
		Load(string) string
	}

	Cache interface {
		GetWebM(string, Loader) string
	}

	Request struct {
		URL string
		C   chan string
	}
)

type result struct {
	Server   string `json:"server"`
	Filename string `json:"filename"`
	State    string `json:"state"`
}

type context struct {
	C     chan Request
	cache Cache
}

func Converter(workers int, retries int) (chan<- Request, util.Handle) {
	ctx := context{
		C:     make(chan Request, 100),
		cache: make(map[string]chan string),
		mu:    new(sync.Mutex),
	}

	for i := 0; i < workers; i++ {
		hs[i] = worker(c, 2*i+3)
	}

	return c, util.MultiHandle(hs...)
}

func worker(ctx context, srv int) util.Handle {
	h := util.NewHandle()
	go func() {
		defer h.Reply()
		for {
			select {
			case req := <-c:
				go handleRequest(ctx, req)

			case <-h.C:
				return
			}
		}
	}()

	return h
}

func handleRequest(ctx context, req Request) {

}
