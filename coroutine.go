package co

import (
	"context"
	"fmt"
	"time"

	"github.com/lsg2020/goco/internal/debug"
	"github.com/lsg2020/goco/internal/logger"
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
	debug  *debug.Debug
	logger logger.Log
}

func (co *Coroutine) RunAsync(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	opts = opts.init(co.opts.OpenDebug)
	return co.ex.Run(ctx, &coTask{ctx: ctx, co: co, f: f, opts: opts})
}

func (co *Coroutine) RunSync(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	opts = opts.init(co.opts.OpenDebug)

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
	sessionID := co.ex.PrepareWait()
	return sessionID
}

func (co *Coroutine) Wait(ctx context.Context, sessionID uint64) error {
	return co.ex.Wait(ctx, sessionID)
}

func (co *Coroutine) Wakeup(sessionID uint64, result error) {
	co.ex.Wakeup(sessionID, result)
}

func (co *Coroutine) Await(ctx context.Context, f TaskFunc) error {
	sessionID := co.PrepareWait()

	err := co.Async(func(ctx context.Context) error {
		err := f(ctx)
		co.Wakeup(sessionID, err)
		return err
	})
	if err != nil {
		co.Wakeup(sessionID, fmt.Errorf("post async task failed, %w", err))
		// return err
	}
	err = co.ex.Wait(ctx, sessionID)
	if err != nil {
		co.ex.Wakeup(sessionID, err)
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
	sessionID := co.ex.PrepareWait()
	time.AfterFunc(d, func() {
		co.ex.Wakeup(sessionID, nil)
	})
	err := co.ex.Wait(ctx, sessionID)
	if err != nil {
		co.ex.Wakeup(sessionID, err)
		return
	}
}

func (co *Coroutine) Close() {
	co.cancel()
}

func (co *Coroutine) init(opts *Options) error {
	if opts.OpenDebug {
		d, err := debug.NewDebug(opts.DebugInfo)
		if err != nil {
			return fmt.Errorf("create debug failed, %w", err)
		}
		co.debug = d
	}
	if opts.Parent != nil {
		co.ctx, co.cancel = context.WithCancel(opts.Parent.ctx)
		co.ex = opts.Parent.ex
		co.logger = opts.Parent.logger
		return nil
	}

	l, err := logger.NewLogger("goco", logger.StringToLevel(co.opts.LogLevel))
	if err != nil {
		return fmt.Errorf("create logger failed, %w", err)
	}
	if co.opts.Logger != nil {
		l.SetLogger(co.opts.Logger, logger.StringToLevel(co.opts.LogLevel))
	}
	co.logger = l

	co.ctx, co.cancel = context.WithCancel(context.Background())
	ex, err := NewExecuter(co.ctx, co.opts.InitWorkAmount, co.opts.WorkChannelSize)
	if err != nil {
		return fmt.Errorf("create executer failed, %w", err)
	}
	co.ex = ex
	return nil
}
