package co

import (
	"runtime"
	"time"
)

// Logger is used for logging formatted messages.
type Logger interface {
	// Printf must have the same semantics as log.Printf.
	Printf(format string, args ...interface{})
}

const (
	defaultInitWorkAmount  = 5
	defaultWorkChannelSize = 1024
	defaultRunLimitTime    = time.Second * 5
)

type Options struct {
	Name   string
	Parent *Coroutine

	Logger   Logger
	LogLevel string

	InitWorkAmount  int
	WorkChannelSize int
	RunLimitTime    time.Duration
	AsyncTaskSubmit func(func()) error

	OpenDebug bool
	DebugInfo string
}

func (opts *Options) init() error {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Name == "" {
		return ErrNeedCoroutineName
	}
	if opts.InitWorkAmount == 0 {
		opts.InitWorkAmount = defaultInitWorkAmount
	}
	if opts.WorkChannelSize == 0 {
		opts.WorkChannelSize = defaultWorkChannelSize
	}
	if opts.RunLimitTime == 0 {
		opts.RunLimitTime = defaultRunLimitTime
	}
	if opts.LogLevel == "" {
		opts.LogLevel = "debug"
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

func (opts *RunOptions) init(debug bool) *RunOptions {
	if opts == nil {
		opts = &RunOptions{}
	}
	if opts.Name == "" {
		opts.Name = "unknown"
	}
	if debug {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			opts.file, opts.line = file, line
		}
	}
	return opts
}
