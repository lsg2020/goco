package co

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

func New(opts *Options) (*Coroutine, error) {
	err := opts.init()
	if err != nil {
		return nil, fmt.Errorf("init options failed, %w", err)
	}

	co := &Coroutine{
		opts: opts,
	}
	err = co.init(opts)
	if err != nil {
		return nil, fmt.Errorf("init coroutine failed, %w", err)
	}
	return co, nil
}

type TaskFunc func(ctx context.Context) error

type Coroutine struct {
	opts   *Options
	ex     *Executer
	ctx    context.Context
	cancel context.CancelFunc

	running int64
	waiting int64
}

func (co *Coroutine) Run(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	return co.ex.Run(ctx, f, co, opts)
}

func (co *Coroutine) RunWait(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	if opts == nil {
		opts = &RunOptions{}
	}
	ch := make(chan error, 1)
	opts.Result = func(err error) {
		ch <- err
	}
	err := co.Run(ctx, f, opts)
	if err != nil {
		return fmt.Errorf("coroutine run failed, %w", err)
	}
	return <-ch
}

func (co *Coroutine) Await(ctx context.Context, f TaskFunc) error {
	sessionID := co.ex.PreWait()

	err := co.Async(func(ctx context.Context) error {
		err := f(ctx)
		co.ex.wakeup(sessionID, err)
		return err
	})
	if err != nil {
		co.ex.wakeup(sessionID, fmt.Errorf("post async task failed, %w", err))
		return err
	}
	err = co.ex.Wait(ctx, sessionID)
	if err != nil {
		co.ex.wakeup(sessionID, err)
		return err
	}
	return nil
}

func (co *Coroutine) Async(f TaskFunc) error {
	task := func() {
		ctx := WithContextCO(co.ctx, co)
		_ = f(ctx)
	}
	if co.opts.AsyncTaskSubmit != nil {
		return co.opts.AsyncTaskSubmit(task)
	}
	go task()
	return nil
}

func (co *Coroutine) Sleep(ctx context.Context, d time.Duration) {
	sessionID := co.ex.PreWait()
	time.AfterFunc(d, func() {
		co.ex.wakeup(sessionID, nil)
	})
	err := co.ex.Wait(ctx, sessionID)
	if err != nil {
		co.ex.wakeup(sessionID, err)
		return
	}
}

func (co *Coroutine) Close() {
	co.cancel()
}

func (co *Coroutine) init(opts *Options) error {
	if opts.Parent != nil {
		co.ctx, co.cancel = context.WithCancel(opts.Parent.ctx)
		co.ex = opts.Parent.ex
		return nil
	}

	co.ctx, co.cancel = context.WithCancel(context.Background())
	ex, err := NewExecuter(co.ctx, co.opts.InitWorkAmount, co.opts.WorkChannelSize)
	if err != nil {
		return fmt.Errorf("create executer failed, %w", err)
	}
	co.ex = ex
	return nil
}

func (co *Coroutine) GetRunning() int64 {
	return atomic.LoadInt64(&co.running)
}

func (co *Coroutine) addRunning(v int64) {
	atomic.AddInt64(&co.running, v)
}

func (co *Coroutine) GetWaiting() int64 {
	return atomic.LoadInt64(&co.waiting)
}

func (co *Coroutine) addWaiting(v int64) {
	atomic.AddInt64(&co.waiting, v)
}
