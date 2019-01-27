package schedx

import (
	"sync"
	"time"
)

type Job struct {
	id       interface{}
	interval time.Duration
	timer    *time.Timer
	mu       sync.Mutex
}

func newJob(id interface{}, interval time.Duration) *Job {
	return &Job{id, interval, nil, sync.Mutex{}}
}

func (j *Job) start(action Action) {
	j.mu.Lock()
	j.scheduleUnsafe(action, 0)
	j.mu.Unlock()
}

func (j *Job) reschedule(action Action) {
	j.mu.Lock()
	if j.timer != nil {
		j.scheduleUnsafe(action, j.interval)
	}

	j.mu.Unlock()
}

func (j *Job) scheduleUnsafe(action Action, interval time.Duration) {
	j.timer = time.AfterFunc(interval, func() {
		action(j.id)
		j.reschedule(action)
	})
}

func (j *Job) cancel() {
	j.mu.Lock()
	if j.timer != nil {
		j.timer.Stop()
		j.timer = nil
	}

	j.mu.Unlock()
}
