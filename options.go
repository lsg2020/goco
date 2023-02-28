package co

import "time"

const (
	defaultInitWorkAmount  = 5
	defaultWorkChannelSize = 1024
	defaultRunLimitTime    = time.Second * 5
)

type Options struct {
	Name            string
	Parent          *Coroutine
	AsyncTaskSubmit func(func()) error

	InitWorkAmount  int
	WorkChannelSize int
	RunLimitTime    time.Duration
	// TaskLimitAmount int32

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

	return nil
}

type RunOptions struct {
	Result       func(error)
	RunLimitTime time.Duration
}
