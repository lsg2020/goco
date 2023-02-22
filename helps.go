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

func FromContextStatus(ctx context.Context) StatusType {
	s, ok := ctx.Value(ctxStatusKey).(StatusType)
	if !ok {
		return StatusEmpty
	}
	return s
}

func WithContextStatus(ctx context.Context, s StatusType) context.Context {
	return context.WithValue(ctx, ctxStatusKey, s)
}

func Run(ctx context.Context, f TaskFunc, opts *RunOptions) error {
	co := FromContextCO(ctx)
	if co == nil {
		return ErrNeedFromCoroutine
	}
	return co.Run(ctx, f, opts)
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
