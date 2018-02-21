package webm

import (
	"github.com/jfk9w/hikkabot/util"
)

const (
	NotFound = "-"
	Pending  = "*"
	Marked   = "="
)

type (
	Client interface {
		Load(string, string) (string, error)
	}

	Cache interface {
		GetWebm(string) string
		UpdateWebm(string, string, string) bool
	}

	Request struct {
		URL string
		C   chan string
	}
)

func NewRequest(url string) Request {
	return Request{
		URL: url,
		C:   make(chan string, 1),
	}
}

func Converter(client Client, cache Cache,
	workers int, retries int) (chan<- Request, util.Handle) {

	c := make(chan Request, 100)
	hs := make([]util.Handle, workers)
	for i := 0; i < workers; i++ {
		hs[i] = worker(&context{
			C:       c,
			client:  client,
			cache:   cache,
			srv:     2*i + 3,
			retries: retries,
		})
	}

	return c, util.MultiHandle(hs...)
}
