package co

import (
	"context"
	"fmt"
	"time"
)

func New(opts *Options) (*Coroutine, error) {
	if opts == nil {
		opts = &Options{}
	}

	co := &Coroutine{
		opts: opts,
	}
	err := co.init(opts)
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
}

func (co *Coroutine) Run(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	return co.ex.Run(ctx, f, co)
}

func (co *Coroutine) Await(ctx context.Context, f TaskFunc) error {
	if FromContextStatus(ctx) != StatusRunning {
		return ErrNeedFromCoroutine
	}

	sessionID := co.ex.PreWait()

	err := co.Async(func(ctx context.Context) error {
		err := f(ctx)
		co.ex.wakeup(sessionID, err)
		return err
	})
	if err != nil {
		co.ex.wakeup(sessionID, err)
		return err
	}
	return co.ex.Wait(ctx, sessionID)
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
	if FromContextStatus(ctx) != StatusRunning {
		return
	}

	sessionID := co.ex.PreWait()
	time.AfterFunc(d, func() {
		co.ex.wakeup(sessionID, nil)
	})
	_ = co.ex.Wait(ctx, sessionID)
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
	ex, err := NewExecuter(co.ctx)
	if err != nil {
		return fmt.Errorf("create executer failed, %w", err)
	}
	co.ex = ex
	return nil
}
