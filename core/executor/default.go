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
	if _, ok := e.getRunningTask(id); ok {
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
		defer e.mu.Lock().Unlock()
		delete(e.tasks, id)
		defer log.Debugf("completed: %s", err)
	})

	e.tasks[id] = rt
}

func (e *Default) getRunningTask(id interface{}) (*runningTask, bool) {
	defer e.mu.RLock().Unlock()
	task, ok := e.tasks[id]
	return task, ok
}

func (e *Default) Cancel(id interface{}) {
	rt, ok := e.getRunningTask(id)
	if !ok {
		return
	}

	rt.cancelAndWait()
}

func (e *Default) Close() error {
	unlocker := e.mu.RLock()
	ids := make([]interface{}, len(e.tasks))
	i := 0
	for id := range e.tasks {
		ids[i] = id
		i++
	}

	unlocker.Unlock()
	for _, id := range ids {
		e.Cancel(id)
	}

	return nil
}
