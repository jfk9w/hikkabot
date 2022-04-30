package core

import (
	"context"
	"fmt"

	"hikkabot/feed"

	"github.com/jfk9w-go/flu/colf"

	"github.com/jfk9w-go/flu/apfel"
	"github.com/jfk9w-go/flu/logf"
	"github.com/jfk9w-go/flu/syncf"
)

const taskExecutorServiceID = "core.task-executor"

type TaskExecutor[C any] struct {
	feed.TaskExecutor
}

func (e TaskExecutor[C]) String() string {
	return taskExecutorServiceID
}

func (e *TaskExecutor[C]) Include(ctx context.Context, app apfel.MixinApp[C]) error {
	if e.TaskExecutor != nil {
		return nil
	}

	executor := &taskExecutor{tasks: make(colf.Set[string])}
	executor.ctx, executor.cancel = context.WithCancel(context.Background())

	if err := app.Manage(ctx, executor); err != nil {
		return err
	}

	e.TaskExecutor = executor
	return nil
}

type taskExecutor struct {
	ctx    context.Context
	tasks  colf.Set[string]
	cancel func()
	work   syncf.WaitGroup
	mu     syncf.RWMutex
}

func (e *taskExecutor) String() string {
	return taskExecutorServiceID
}

func (e *taskExecutor) Submit(id any, task feed.Task) {
	key := fmt.Sprint(id)
	ctx, cancel := e.mu.Lock(e.ctx)
	if ctx.Err() != nil {
		return
	}

	defer cancel()
	if _, ok := e.tasks[key]; ok {
		return
	}

	_, _ = syncf.GoWith(e.ctx, e.work.Spawn, func(ctx context.Context) {
		defer func() {
			_, cancel := e.mu.Lock(context.Background())
			defer cancel()
			delete(e.tasks, key)
		}()

		err := task(ctx)
		logf.Resultf(ctx, logf.Debug, logf.Warn, "task [%s] completed: %v", key, err)
	})

	e.tasks.Add(key)
	logf.Get(e).Debugf(ctx, "started task [%s]", key)
}

func (e *taskExecutor) Close() error {
	e.cancel()
	e.work.Wait()
	return nil
}
