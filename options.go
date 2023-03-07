package co

import (
	"context"
	"runtime"
	"time"
)

const (
	defaultInitWorkAmount  = 5
	defaultWorkChannelSize = 1024
	defaultRunLimitTime    = time.Second * 5
)

type ExOptions struct {
	Name            string
	InitWorkAmount  int
	WorkChannelSize int
	AsyncTaskSubmit func(func()) error

	HookRun  func(ex *Executer, task Task, ctx context.Context)
	HookWait func(ex *Executer, t Task, sessionID uint64, f func() error)

	OnWorkerCreate func(ex *Executer)
	OnTaskStart    func(ex *Executer, t Task)
	OnTaskRunning  func(ex *Executer, t Task)
	OnTaskFinish   func(ex *Executer, t Task)
	OnWakeup       func(ex *Executer, ok bool, result error)
}

func (opts *ExOptions) init() error {
	if opts == nil {
		opts = &ExOptions{}
	}
	if opts.Name == "" {
		return ErrNeedExecuterName
	}
	if opts.InitWorkAmount == 0 {
		opts.InitWorkAmount = defaultInitWorkAmount
	}
	if opts.WorkChannelSize == 0 {
		opts.WorkChannelSize = defaultWorkChannelSize
	}

	return nil
}

type Options struct {
	Name         string
	DebugInfo    string
	RunLimitTime time.Duration
	Executer     *Executer

	OnTaskSuspended func(co *Coroutine, t Task)
	OnTaskResume    func(co *Coroutine, t Task)
	OnTaskRunning   func(co *Coroutine, t Task)
	OnTaskFinish    func(co *Coroutine, t Task)
	OnTaskRecover   func(co *Coroutine, t Task, err error)
	OnTaskTimeout   func(co *Coroutine, t Task)
}

func (opts *Options) init() error {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Name == "" {
		return ErrNeedCoroutineName
	}
	if opts.Executer == nil {
		return ErrNeedExecuter
	}
	if opts.RunLimitTime == 0 {
		opts.RunLimitTime = defaultRunLimitTime
	}
	return nil
}

type RunOptions struct {
	Name         string
	Result       func(error)
	RunLimitTime time.Duration

	file string
	line int
}

func (opts *RunOptions) init() *RunOptions {
	if opts == nil {
		opts = &RunOptions{}
	}
	if opts.Name == "" {
		opts.Name = "unknown"
	}
	_, file, line, ok := runtime.Caller(2)
	if ok {
		opts.file, opts.line = file, line
	}
	return opts
}
