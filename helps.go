package co

import (
	"context"
	"time"
)

func FromContextCO(ctx context.Context) *Coroutine {
	co, ok := ctx.Value(ctxCOKey).(*Coroutine)
	if !ok {
		return nil
	}
	return co
}

func WithContextCO(ctx context.Context, co *Coroutine) context.Context {
	return context.WithValue(ctx, ctxCOKey, co)
}

func FromContextTask(ctx context.Context) Task {
	s, ok := ctx.Value(ctxTaskKey).(Task)
	if !ok {
		return nil
	}
	return s
}

func WithContextTask(ctx context.Context, task Task) context.Context {
	return context.WithValue(ctx, ctxTaskKey, task)
}

func RunAsync(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.RunAsync(ctx, f, opts)
}

func RunSync(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.RunSync(ctx, f, opts)
}

func PrepareWait(ctx context.Context) (uint64, error) {
	co := FromContextCO(ctx)
	if co == nil {
		return 0, ErrNeedFromCoroutine
	}
	sessionID := co.PrepareWait()
	return sessionID, nil
}

func Wait(ctx context.Context, sessionID uint64) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.Wait(ctx, sessionID)
}

func Wakeup(ctx context.Context, sessionID uint64, result error) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	co.ex.Wakeup(sessionID, result)
	return nil
}

func Await(ctx context.Context, f TaskFunc) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.Await(ctx, f)
}

func Async(ctx context.Context, f TaskFunc) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.Async(f)
}

func Sleep(ctx context.Context, d time.Duration) {
	co := FromContextCO(ctx)
	if co == nil {
		time.Sleep(d)
		return
	}
	co.Sleep(ctx, d)
}
