package co

import "context"

type Task struct {
	ctx    context.Context
	co     *Coroutine
	status StatusType

	f    TaskFunc
	opts *RunOptions

	// wakeup
	wakeup    bool
	sessionID uint64
	err       error
}

func (t *Task) OnResult(err error) {
	if t.opts != nil && t.opts.Result != nil {
		t.opts.Result(err)
		return
	}
}
