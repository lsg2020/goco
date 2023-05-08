package co

import (
	"context"
)

const (
	defaultInitWorkAmount  = 5
	defaultWorkChannelSize = 1024
	defaultWaitShards      = 5
)

type FilterRunChain func(next FilterRun) FilterRun
type FilterRun func(task Task, ctx context.Context) error
type FilterWaitChain func(next FilterWait) FilterWait
type FilterWait func(ex *Executer, t Task, sessionID uint64, ctx context.Context) error

type ExOptions struct {
	Name            string
	InitWorkAmount  int
	WorkChannelSize int
	AsyncTaskSubmit func(func()) error
	WaitShards      uint64

	HookRun  []FilterRunChain
	HookWait []FilterWaitChain

	OnWorkerCreate func(ex *Executer)
	OnTaskStart    func(ex *Executer, t Task)
	OnTaskRunning  func(ex *Executer, t Task)
	OnTaskFinish   func(ex *Executer, t Task)
	OnWakeup       func(ex *Executer, ok bool, result error)
}

func (opts *ExOptions) init() error {
	if opts.Name == "" {
		return ErrNeedExecuterName
	}
	if opts.InitWorkAmount == 0 {
		opts.InitWorkAmount = defaultInitWorkAmount
	}
	if opts.WorkChannelSize == 0 {
		opts.WorkChannelSize = defaultWorkChannelSize
	}
	if opts.WaitShards == 0 {
		opts.WaitShards = defaultWaitShards
	}

	return nil
}

type Options struct {
	Name     string
	Executer *Executer

	OnTaskSuspended func(co *Coroutine, t Task)
	OnTaskResume    func(co *Coroutine, t Task)
	OnTaskRunning   func(co *Coroutine, t Task)
	OnTaskFinish    func(co *Coroutine, t Task)
	OnTaskRecover   func(co *Coroutine, t Task, err error)
	OnTaskTimeout   func(co *Coroutine, t Task)
}

func (opts *Options) init() error {
	if opts.Name == "" {
		return ErrNeedCoroutineName
	}
	if opts.Executer == nil {
		return ErrNeedExecuter
	}
	return nil
}

type RunOptions struct {
	Name    string
	Result  func(error)
	HookRun []FilterRunChain
}

func (opts *RunOptions) init() *RunOptions {
	if opts == nil {
		opts = &RunOptions{}
	}
	if opts.Name == "" {
		opts.Name = "unknown"
	}
	return opts
}
