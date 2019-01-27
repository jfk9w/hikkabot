package aconvert

import (
	"sync"
	"testing"

	"github.com/jfk9w-go/hikkabot/common/httpx"
)

func TestBalancer_Convert(t *testing.T) {
	var (
		file     = &httpx.File{Path: "testdata/15301775501121.webm"}
		balancer = ConfigureBalancer(Config{
			Retries: 1,
			Http: &httpx.Config{
				Transport: &httpx.TransportConfig{
					Log: "httpx",
				},
			},
		})

		group sync.WaitGroup
		count = 7
	)

	group.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			defer group.Done()
			_, err := balancer.Convert(file)
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}

	group.Wait()
	balancer.Close()
}
