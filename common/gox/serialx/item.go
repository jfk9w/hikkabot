package serialx

import (
	"time"
)

type Status uint8

const (
	Ok Status = iota
	Failed
	Delay
)

type Out interface {
	Status() Status
	Delay() time.Duration
}

type Action func(interface{}) Out

type Item struct {
	action Action
	retry  int
	out    chan Out
}

func NewItem(action Action, retry int) *Item {
	return &Item{
		action: action,
		retry:  retry,
		out:    make(chan Out, 1),
	}
}

func (i *Item) Resolve(in interface{}) bool {
	out := i.action(in)

	switch out.Status() {
	case Delay:
		delay := out.Delay()
		time.Sleep(delay)
		return false

	case Failed:
		if i.retry > 0 {
			i.retry--
			return false
		}
	}

	i.out <- out
	return true
}

func (i *Item) Out() Out {
	return <-i.out
}
