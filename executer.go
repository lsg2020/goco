package co

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type wakeup struct {
	sessionID uint64
	result    error
}

type task struct {
	ctx context.Context
	t   Task
}

func NewExecuter(ctx context.Context, opts *ExOptions) (*Executer, error) {
	err := opts.init()
	if err != nil {
		return nil, fmt.Errorf("executer options init failed, %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	ex := &Executer{
		ctx:    ctx,
		cancel: cancel,
		opts:   opts,
	}
	err = ex.start(ctx, opts.InitWorkAmount, opts.WorkChannelSize)
	if err != nil {
		return nil, fmt.Errorf("executer start failed, %w", err)
	}
	return ex, nil
}

type Executer struct {
	ctx    context.Context
	cancel context.CancelFunc
	opts   *ExOptions

	cond *sync.Cond

	workID      int
	workAmount  int
	waitAmount  int
	nextSession uint64

	tasks   chan task
	wakeups chan wakeup

	waitConds  map[uint64]*sync.Cond
	waitResult error

	waitSessionMgr *WaitSessionMgr
}

func (ex *Executer) GetOpts() *ExOptions {
	return ex.opts
}

func (ex *Executer) GetCtx() context.Context {
	return ex.ctx
}

func (ex *Executer) Close() {
	ex.cancel()
}

func (ex *Executer) Run(ctx context.Context, t Task) error {
	ex.OnTaskStart(t)
	select {
	case ex.tasks <- task{ctx: ctx, t: t}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ex *Executer) Wakeup(sessionID uint64, err error) {
	ex.wakeups <- wakeup{sessionID: sessionID, result: err}
}

func (ex *Executer) PrepareWait() uint64 {
	sessionID := atomic.AddUint64(&ex.nextSession, 1)
	if sessionID == 0 {
		sessionID = atomic.AddUint64(&ex.nextSession, 1)
	}
	ex.waitConds[sessionID] = sync.NewCond(ex.cond.L)
	return sessionID
}

func (ex *Executer) Wait(ctx context.Context, sessionID uint64) error {
	t := FromContextTask(ctx)
	if t == nil {
		panic(fmt.Errorf("wait need in coroutine, %w", ErrNeedFromCoroutine))
	}

	waitCond, ok := ex.waitConds[sessionID]
	if !ok {
		panic(fmt.Errorf("wait session not found, %w", ErrWaitSessionMiss))
	}
	ex.waitAmount++
	currentWorkID := ex.workID

	if ex.waitAmount == ex.workAmount {
		ex.workAmount++
		initWg := &sync.WaitGroup{}
		initWg.Add(1)
		go ex.work(initWg, ex.workAmount)

		ex.cond.L.Unlock()
		initWg.Wait()
		ex.cond.L.Lock()
	}

	impl := func(ex *Executer, t Task, sessionID uint64, ctx context.Context) error {
		t.OnSuspended()
		ex.waitSessionMgr.PushWait(sessionID, ctx)

		ex.cond.Signal()
		waitCond.Wait()

		ex.waitSessionMgr.PopWait(sessionID)
		t.OnResume()
		return ex.waitResult
	}
	for i := len(ex.opts.HookWait) - 1; i >= 0; i-- {
		impl = ex.opts.HookWait[i](impl)
	}
	r := impl(ex, t, sessionID, ctx)

	ex.workID = currentWorkID
	ex.waitAmount--
	// delete(ex.waitConds, sessionID)

	return r
}

func (ex *Executer) start(ctx context.Context, initWorkAmount int, channelSize int) error {
	ex.cond = sync.NewCond(new(sync.Mutex))
	ex.nextSession = 0

	ex.ctx = ctx
	ex.tasks = make(chan task, channelSize)
	ex.wakeups = make(chan wakeup, channelSize)

	ex.waitConds = make(map[uint64]*sync.Cond)
	ex.waitSessionMgr = NewWaitSessionMgr(ctx, ex.opts.WaitShards, func(sessionID uint64) {
		ex.Wakeup(sessionID, ErrCancelContext)
	})

	initWg := &sync.WaitGroup{}
	ex.workAmount = initWorkAmount
	for i := 0; i < initWorkAmount; i++ {
		initWg.Add(1)
		go ex.work(initWg, i+1)
	}
	initWg.Wait()
	ex.cond.L.Lock()
	ex.cond.Signal()
	ex.cond.L.Unlock()
	return nil
}

func (ex *Executer) work(initWg *sync.WaitGroup, workID int) {
	ex.cond.L.Lock()
	ex.OnWorkerCreate()
	initWg.Done()
	ex.cond.Wait()

	run := func(task Task, ctx context.Context) error {
		return task.Run(ctx)
	}

	ex.workID = workID
	for {
		select {
		case info := <-ex.tasks:
			ex.OnTaskRunning(info.t)
			impl := run
			for i := len(ex.opts.HookRun) - 1; i >= 0; i-- {
				impl = ex.opts.HookRun[i](impl)
			}
			info.t.OnResult(impl(info.t, info.ctx))
			ex.OnTaskFinish(info.t)
		case w := <-ex.wakeups:
			waitCond, ok := ex.waitConds[w.sessionID]
			ex.OnWakeup(ok, w.result)
			if ok {
				delete(ex.waitConds, w.sessionID)
				ex.waitResult = w.result
				waitCond.Signal()
				ex.cond.Wait()
				ex.workID = workID
			}
		case <-ex.ctx.Done():
			return
		}
	}
}

func (ex *Executer) OnWorkerCreate() {
	if ex.opts.OnWorkerCreate != nil {
		ex.opts.OnWorkerCreate(ex)
	}
}

func (ex *Executer) OnTaskStart(t Task) {
	if ex.opts.OnTaskStart != nil {
		ex.opts.OnTaskStart(ex, t)
	}
}

func (ex *Executer) OnTaskRunning(t Task) {
	if ex.opts.OnTaskRunning != nil {
		ex.opts.OnTaskRunning(ex, t)
	}
}

func (ex *Executer) OnTaskFinish(t Task) {
	if ex.opts.OnTaskFinish != nil {
		ex.opts.OnTaskFinish(ex, t)
	}
}

func (ex *Executer) OnWakeup(ok bool, result error) {
	if ex.opts.OnWakeup != nil {
		ex.opts.OnWakeup(ex, ok, result)
	}
}
