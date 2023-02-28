package co

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

type StatusType int32

const (
	StatusDead      StatusType = 0
	StatusSuspended StatusType = 1
	StatusRunning   StatusType = 2
)

type coTask struct {
	ctx    context.Context
	co     *Coroutine
	status int32

	f    TaskFunc
	opts *RunOptions

	limitTimer *time.Timer
}

func (t *coTask) OnSuspended() {
	t.setStatus(StatusSuspended)
	t.co.debug.AddWaiting(1)
}

func (t *coTask) OnResume() {
	t.co.debug.AddWaiting(-1)
	t.setStatus(StatusRunning)
}

func (t *coTask) OnResult(err error) {
	if t.opts != nil && t.opts.Result != nil {
		t.opts.Result(err)
		return
	}
}

func (t *coTask) Run() {
	ctx := WithContextCO(t.ctx, t.co)
	ctx = WithContextTask(ctx, t)

	defer func() {
		if t.limitTimer != nil {
			t.limitTimer.Stop()
		}
		t.setStatus(StatusDead)
		t.co.debug.AddRunning(-1)
		if r := recover(); r != nil {
			t.OnResult(fmt.Errorf("task panic, %v", r))
		}
	}()

	t.setStatus(StatusRunning)
	t.co.debug.AddRunning(1)
	t.checkRunLimitTime()

	r := t.f(ctx)
	t.OnResult(r)
}

func (t *coTask) setStatus(s StatusType) {
	atomic.StoreInt32(&t.status, int32(s))
}

func (t *coTask) getStatus() StatusType {
	return StatusType(atomic.LoadInt32(&t.status))
}

func (t *coTask) getRunLimitTime() time.Duration {
	if t.opts.RunLimitTime < 0 {
		return 0
	} else if t.opts.RunLimitTime == 0 {
		if t.co.opts.RunLimitTime < 0 {
			return 0
		}
		return t.co.opts.RunLimitTime
	} else {
		return t.opts.RunLimitTime
	}
}

func (t *coTask) checkRunLimitTime() {
	limitTime := t.getRunLimitTime()
	if limitTime <= 0 {
		return
	}
	t.limitTimer = time.AfterFunc(limitTime, func() {
		if t.getStatus() != StatusDead {
			fmt.Println("---------------- run limit time", t.co.debug)
		}
	})
}
