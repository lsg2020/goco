package co

import (
	"context"
	"fmt"
	"sync/atomic"
)

type StatusType int32

const (
	StatusDead      StatusType = 0
	StatusSuspended StatusType = 1
	StatusRunning   StatusType = 2
)

type Task interface {
	GetName() string
	Run(context.Context) error
	OnResult(err error)
	OnSuspended()
	OnResume()
}

type coTask struct {
	co     *Coroutine
	status int32

	f    TaskFunc
	opts *RunOptions
}

func (t *coTask) OnSuspended() {
	t.co.OnTaskSuspended(t)
	t.setStatus(StatusSuspended)
}

func (t *coTask) OnResume() {
	t.setStatus(StatusRunning)
	t.co.OnTaskResume(t)
}

func (t *coTask) Run(ctx context.Context) error {
	ctx = WithContextCO(ctx, t.co)
	ctx = WithContextTask(ctx, t)

	impl := func(ctx context.Context) (err error) {
		t.co.OnTaskRunning(t)

		defer func() {
			t.setStatus(StatusDead)
			t.co.OnTaskFinish(t)
			if r := recover(); r != nil {
				if !t.co.IsClose() || r != ErrCoroutineClosed {
					err = fmt.Errorf("task panic, %v", r)
					t.co.OnTaskRecover(t, err)
					panic(err)
				}
			}
		}()

		t.setStatus(StatusRunning)
		err = t.f(ctx)
		return
	}

	if t.opts.HookRun != nil {
		return t.opts.HookRun(t, ctx, impl)
	} else {
		return impl(ctx)
	}
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

func (t *coTask) GetName() string {
	name := t.co.opts.Name + ":" + t.co.opts.DebugInfo + ":" + t.opts.Name
	return name
}
