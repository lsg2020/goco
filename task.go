package co

import (
	"context"
	"fmt"
	"strconv"
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
	co     *Coroutine
	status int32

	f    TaskFunc
	opts *RunOptions

	limitTimer *time.Timer
}

func (t *coTask) OnSuspended() {
	t.co.OnTaskSuspended(t)
	t.setStatus(StatusSuspended)
}

func (t *coTask) OnResume() {
	t.setStatus(StatusRunning)
	t.co.OnTaskResume(t)
}

func (t *coTask) Run(ctx context.Context) {
	t.co.OnTaskRunning(t)
	ctx = WithContextCO(ctx, t.co)
	ctx = WithContextTask(ctx, t)

	defer func() {
		if t.limitTimer != nil {
			t.limitTimer.Stop()
		}
		t.setStatus(StatusDead)
		t.co.OnTaskFinish(t)
		if r := recover(); r != nil {
			if !t.co.IsClose() {
				err := fmt.Errorf("task panic, %v", r)
				t.co.OnTaskRecover(t, err)
				t.OnResult(err)
			}
		}
	}()

	t.setStatus(StatusRunning)
	t.checkRunLimitTime()

	r := t.f(ctx)
	t.OnResult(r)
}

func (t *coTask) OnResult(err error) {
	if t.opts.Result != nil {
		t.opts.Result(err)
		return
	}
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
			t.co.OnTaskTimeout(t)
		}
	})
}

func (t *coTask) GetName() string {
	name := t.co.opts.Name + ":" + t.co.opts.DebugInfo + ":" + t.opts.Name
	if t.opts.line != 0 {
		name += "( " + t.opts.file + ":" + strconv.FormatInt(int64(t.opts.line), 10) + " )"
	}
	return name
}
