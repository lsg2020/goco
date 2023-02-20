package co

import "errors"

var (
	ErrCancelContext        = errors.New("coroutine cancel context")
	ErrWaitSessionMiss      = errors.New("wait session miss")
	ErrCoroutineLimitAmount = errors.New("coroutine limit amount")
	ErrNeedFromCoroutine    = errors.New("need from coroutine")
	ErrAlreadyInCoroutine   = errors.New("already in coroutine")
)

type ContextKey struct{ _ int }

var (
	ctxCOKey     = &ContextKey{}
	ctxStatusKey = &ContextKey{}
)

type StatusType int

const (
	StatusEmpty   StatusType = 0
	StatusRunning StatusType = 1
)
