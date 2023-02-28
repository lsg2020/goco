package co

import "errors"

var (
	ErrNeedCoroutineName    = errors.New("coroutine need name")
	ErrCancelContext        = errors.New("coroutine cancel context")
	ErrWaitSessionMiss      = errors.New("waiting session miss")
	ErrCoroutineLimitAmount = errors.New("coroutine limit amount")
	ErrNeedFromCoroutine    = errors.New("need from coroutine")
	ErrAlreadyInCoroutine   = errors.New("already in coroutine")
)

type ContextKey struct{ _ int }

var (
	ctxCOKey   = &ContextKey{}
	ctxTaskKey = &ContextKey{}
)
