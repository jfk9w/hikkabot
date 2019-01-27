package schedx

import (
	"time"

	"github.com/jfk9w-go/hikkabot/common/gox/syncx"
)

type Action func(interface{})

type T struct {
	jobs     syncx.Map
	interval time.Duration
	action   Action
}

func New(interval time.Duration) *T {
	return &T{
		jobs:     syncx.NewMap(),
		interval: interval,
	}
}

func (scheduler *T) Init(action Action) {
	scheduler.action = action
}

func (scheduler *T) Schedule(id interface{}) {
	var (
		any interface{}
		ok  bool
		job *Job
	)

	any, ok = scheduler.jobs.ComputeIfAbsent(id, func() interface{} {
		return newJob(id, scheduler.interval)
	})

	job = any.(*Job)
	if ok {
		job.start(scheduler.action)
	}
}

func (scheduler *T) Cancel(id interface{}) bool {
	var (
		any interface{}
		ok  bool
		job *Job
	)

	any, ok = scheduler.jobs.Delete(id)
	if ok {
		job = any.(*Job)
		job.cancel()
	}

	return ok
}
