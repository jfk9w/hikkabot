package executor

import (
	"context"

	"github.com/jfk9w-go/flu"
	"github.com/sirupsen/logrus"
)

type runningTask struct {
	work   flu.WaitGroup
	cancel func()
}

func (t *runningTask) cancelAndWait() {
	t.cancel()
	t.work.Wait()
}

type Default struct {
	ctx    context.Context
	tasks  map[interface{}]*runningTask
	mu     flu.RWMutex
	cancel func()
}

func NewDefault(ctx context.Context) *Default {
	ctx, cancel := context.WithCancel(ctx)
	return &Default{
		ctx:    ctx,
		tasks:  make(map[interface{}]*runningTask),
		cancel: cancel,
	}
}

func (e *Default) Submit(id interface{}, task Task) {
	if e.checkRunningTask(id) {
		return
	}

	defer e.mu.Lock().Unlock()
	_, ok := e.tasks[id]
	if ok {
		return
	}

	log := logrus.WithField("task", id)
	rt := new(runningTask)
	rt.cancel = rt.work.Go(e.ctx, func(ctx context.Context) {
		err := task.Execute(ctx)
		log.Debugf("completed: %s", err)
	})

	e.tasks[id] = rt
}

func (e *Default) checkRunningTask(id interface{}) bool {
	defer e.mu.RLock().Unlock()
	_, ok := e.tasks[id]
	return ok
}

func (e *Default) Cancel(id interface{}) {
	if !e.checkRunningTask(id) {
		return
	}

	defer e.mu.Lock().Unlock()
	rt, ok := e.tasks[id]
	if !ok {
		return
	}

	rt.cancelAndWait()
	delete(e.tasks, id)
}

func (e *Default) Close() error {
	e.cancel()
	defer e.mu.RLock().Unlock()
	for _, rt := range e.tasks {
		rt.cancelAndWait()
	}

	return nil
}
