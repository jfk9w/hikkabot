package serialx

import (
	"testing"
	"time"
)

type ok struct{}

func (ok) Status() Status {
	return Ok
}

func (ok) Delay() time.Duration {
	return 0
}

func TestT_Submit_Ok(t *testing.T) {
	var (
		count = 30
		delay = 10 * time.Millisecond
		sx    = New(delay, 0, 0)
		times = make(chan time.Time, count)
	)

	for i := 0; i < count; i++ {
		go func() {
			sx.Submit(func(_ interface{}) Out {
				return ok{}
			})

			times <- time.Now()
		}()
	}

	last := <-times
	for i := 0; i < count-1; i++ {
		time := <-times
		if time.Sub(last) < delay {
			t.Fatal()
		}

		last = time
	}
}
