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

func FromContextTask(ctx context.Context) *Task {
	s, ok := ctx.Value(ctxTaskKey).(*Task)
	if !ok {
		return nil
	}
	return s
}

func WithContextTask(ctx context.Context, task *Task) context.Context {
	return context.WithValue(ctx, ctxTaskKey, task)
}

func Run(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.Run(ctx, f, opts)
}

func RunWait(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.RunWait(ctx, f, opts)
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
