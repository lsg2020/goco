package co

import (
	"context"
	"fmt"
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
	opts    *Options
	isClose bool
}

func (co *Coroutine) GetOpts() *Options {
	return co.opts
}

func (co *Coroutine) GetExecuter() *Executer {
	return co.opts.Executer
}

func (co *Coroutine) IsClose() bool {
	return co.isClose
}

func (co *Coroutine) Close() {
	if co.IsClose() {
		return
	}

	co.isClose = true
}

func (co *Coroutine) RunAsync(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	opts = opts.init()
	return co.GetExecuter().Run(ctx, &coTask{co: co, f: f, opts: opts})
}

func (co *Coroutine) RunSync(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	opts = opts.init()

	ch := make(chan error, 1)
	opts.Result = func(err error) {
		ch <- err
	}
	err := co.RunAsync(ctx, f, opts)
	if err != nil {
		return fmt.Errorf("coroutine run failed, %w", err)
	}
	return <-ch
}

func (co *Coroutine) PrepareWait() uint64 {
	sessionID := co.GetExecuter().PrepareWait()
	return sessionID
}

func (co *Coroutine) Wait(ctx context.Context, sessionID uint64) error {
	r := co.GetExecuter().Wait(ctx, sessionID)
	if co.IsClose() {
		panic(ErrCoroutineClosed)
	}
	return r
}

func (co *Coroutine) Wakeup(sessionID uint64, result error) {
	co.GetExecuter().Wakeup(sessionID, result)
}

func (co *Coroutine) Await(ctx context.Context, f TaskFunc) error {
	sessionID := co.PrepareWait()

	err := co.Async(ctx, func(ctx context.Context) error {
		err := f(ctx)
		co.Wakeup(sessionID, err)
		return err
	})
	if err != nil {
		co.Wakeup(sessionID, fmt.Errorf("post async task failed, %w", err))
		// return err
	}
	err = co.Wait(ctx, sessionID)
	if err != nil {
		return err
	}
	return nil
}

func (co *Coroutine) Async(ctx context.Context, f TaskFunc) error {
	t := func() {
		ctx := WithContextCO(ctx, co)
		_ = f(ctx)
	}
	if co.GetExecuter().GetOpts().AsyncTaskSubmit != nil {
		return co.GetExecuter().GetOpts().AsyncTaskSubmit(t)
	}
	go t()
	return nil
}

func (co *Coroutine) Sleep(ctx context.Context, d time.Duration) {
	sessionID := co.GetExecuter().PrepareWait()
	time.AfterFunc(d, func() {
		co.Wakeup(sessionID, nil)
	})
	err := co.Wait(ctx, sessionID)
	if err != nil {
		return
	}
}

func (co *Coroutine) OnTaskSuspended(t Task) {
	if co.opts.OnTaskSuspended != nil {
		co.opts.OnTaskSuspended(co, t)
	}
}

func (co *Coroutine) OnTaskResume(t Task) {
	if co.opts.OnTaskResume != nil {
		co.opts.OnTaskResume(co, t)
	}
}

func (co *Coroutine) OnTaskRunning(t Task) {
	if co.opts.OnTaskRunning != nil {
		co.opts.OnTaskRunning(co, t)
	}
}

func (co *Coroutine) OnTaskFinish(t Task) {
	if co.opts.OnTaskFinish != nil {
		co.opts.OnTaskFinish(co, t)
	}
}

func (co *Coroutine) OnTaskRecover(t Task, err error) {
	if co.opts.OnTaskRecover != nil {
		co.opts.OnTaskRecover(co, t, err)
	}
}

func (co *Coroutine) OnTaskTimeout(t Task) {
	if co.opts.OnTaskTimeout != nil {
		co.opts.OnTaskTimeout(co, t)
	}
}

func (co *Coroutine) init(opts *Options) error {
	return nil
}
