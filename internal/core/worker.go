package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

type TaskPriority int

const (
	PriorityLight TaskPriority = 1
	PriorityHeavy TaskPriority = 2
)

type Task struct {
	Priority TaskPriority
	Handler  func(ctx context.Context) (string, error)
	OnStart  func()
	resultCh chan taskResult
}

type taskResult struct {
	reply string
	err   error
}

type WorkerPool struct {
	lightCh      chan *Task
	heavyCh      chan *Task
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	shuttingDown atomic.Bool
}

func NewWorkerPool(workers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &WorkerPool{
		lightCh: make(chan *Task, 64),
		heavyCh: make(chan *Task, 64),
		ctx:     ctx,
		cancel:  cancel,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.run(i)
	}
	return p
}

func (p *WorkerPool) Submit(ctx context.Context, task *Task) (string, error) {
	if p.shuttingDown.Load() {
		return "", fmt.Errorf("pool is shutting down")
	}
	task.resultCh = make(chan taskResult, 1)

	ch := p.lightCh
	if task.Priority == PriorityHeavy {
		ch = p.heavyCh
	}
	select {
	case ch <- task:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	select {
	case result := <-task.resultCh:
		return result.reply, result.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (p *WorkerPool) Shutdown() {
	p.shuttingDown.Store(true)
	p.cancel()
	p.wg.Wait()
	// 清空队列，对每个被丢弃的任务发送错误响应，避免 Submit 永久挂起
	for {
		select {
		case task := <-p.lightCh:
			if task.resultCh != nil {
				task.resultCh <- taskResult{err: fmt.Errorf("pool shutdown")}
			}
		case task := <-p.heavyCh:
			if task.resultCh != nil {
				task.resultCh <- taskResult{err: fmt.Errorf("pool shutdown")}
			}
		default:
			return
		}
	}
}

func (p *WorkerPool) run(id int) {
	defer p.wg.Done()
	slog.Info("worker started", "id", id)
	for {
		var task *Task
		select {
		case task = <-p.lightCh:
		default:
			select {
			case task = <-p.lightCh:
			case task = <-p.heavyCh:
			case <-p.ctx.Done():
				return
			}
		}
		if task == nil {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("worker panic", "id", id, "panic", r)
					task.resultCh <- taskResult{err: fmt.Errorf("worker panic: %v", r)}
				}
			}()
			if task.OnStart != nil {
				task.OnStart()
			}
			reply, err := task.Handler(p.ctx)
			task.resultCh <- taskResult{reply: reply, err: err}
		}()
	}
}
