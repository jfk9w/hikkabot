package feed

import (
	"context"
	"log"
	"sync"

	"github.com/jfk9w-go/flu"
)

type Task interface {
	Execute(ctx context.Context) error
}

type TaskExecutor interface {
	Submit(id interface{}, task Task)
	Cancel(id interface{})
	Close()
}

type DefaultExecutor struct {
	ctx    context.Context
	cancel func()
	tasks  map[interface{}]func()
	flu.Mutex
	work sync.WaitGroup
}

func NewTaskExecutor() *DefaultExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	return &DefaultExecutor{
		ctx:    ctx,
		cancel: cancel,
		tasks:  make(map[interface{}]func()),
	}
}

func (e *DefaultExecutor) Submit(id interface{}, task Task) {
	defer e.Lock().Unlock()
	if _, ok := e.tasks[id]; ok {
		return
	}

	ctx, cancel := context.WithCancel(e.ctx)
	e.work.Add(1)
	e.tasks[id] = cancel
	log.Printf("[task > %v] started", id)
	go e.execute(ctx, id, task)
}

func (e *DefaultExecutor) execute(ctx context.Context, id interface{}, task Task) {
	defer func() {
		e.Cancel(id)
		e.work.Done()
	}()

	if err := task.Execute(ctx); err != nil {
		log.Printf("[task > %v] %s", id, err)
		return
	}
}

func (e *DefaultExecutor) Cancel(id interface{}) {
	defer e.Lock().Unlock()
	if cancel, ok := e.tasks[id]; ok {
		cancel()
		delete(e.tasks, id)
	}
}

func (e *DefaultExecutor) Close() {
	e.cancel()
	e.work.Wait()
}
