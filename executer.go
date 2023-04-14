package co

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
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

	workId      int
	workAmount  int
	waitAmount  int
	nextSession uint64

	tasks   chan task
	wakeups chan wakeup

	waitConds  map[uint64]*sync.Cond
	waitResult error

	waitMutex    sync.Mutex
	waitContext  context.Context
	waitCancel   context.CancelFunc
	waitSessions map[uint64]context.Context
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
	currentWorkId := ex.workId

	if ex.waitAmount == ex.workAmount {
		ex.workAmount++
		initWg := &sync.WaitGroup{}
		initWg.Add(1)
		go ex.work(initWg, ex.workAmount)

		ex.cond.L.Unlock()
		initWg.Wait()
		ex.cond.L.Lock()
	}

	impl := func(ex *Executer, t Task, sessionID uint64) error {
		t.OnSuspended()
		ex.pushWait(sessionID, ctx)

		ex.cond.Signal()
		waitCond.Wait()

		ex.popWait(sessionID)
		t.OnResume()
		return ex.waitResult
	}
	for i := len(ex.opts.HookWait) - 1; i >= 0; i-- {
		impl = ex.opts.HookWait[i](impl)
	}
	_ = impl(ex, t, sessionID)

	ex.workId = currentWorkId
	ex.waitAmount--
	// delete(ex.waitConds, sessionID)

	return ex.waitResult
}

func (ex *Executer) start(ctx context.Context, initWorkAmount int, channelSize int) error {
	ex.cond = sync.NewCond(new(sync.Mutex))
	ex.nextSession = 0

	ex.ctx = ctx
	ex.tasks = make(chan task, channelSize)
	ex.wakeups = make(chan wakeup, channelSize)

	ex.workAmount = initWorkAmount

	ex.waitContext, ex.waitCancel = context.WithCancel(ctx)
	ex.waitSessions = make(map[uint64]context.Context)
	ex.waitConds = make(map[uint64]*sync.Cond)
	go ex.monitor(ctx)

	initWg := &sync.WaitGroup{}
	for i := 0; i < ex.workAmount; i++ {
		initWg.Add(1)
		go ex.work(initWg, i+1)
	}
	initWg.Wait()
	ex.cond.L.Lock()
	ex.cond.Signal()
	ex.cond.L.Unlock()
	return nil
}

func (ex *Executer) monitor(ctx context.Context) {
	cases := make([]reflect.SelectCase, 0, 128)
	sessions := make([]uint64, 0, 128)
	pushCase := func(cases []reflect.SelectCase, sessions []uint64, sessionID uint64, ctx context.Context) ([]reflect.SelectCase, []uint64) {
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())})
		sessions = append(sessions, sessionID)
		return cases, sessions
	}

	for {
		cases = cases[:0]
		sessions = sessions[:0]

		ex.waitMutex.Lock()
		ex.waitContext, ex.waitCancel = context.WithCancel(ctx)
		cases, sessions = pushCase(cases, sessions, 0, ctx)
		cases, sessions = pushCase(cases, sessions, 0, ex.waitContext)
		for session, ctx := range ex.waitSessions {
			cases, sessions = pushCase(cases, sessions, session, ctx)
			if len(cases) > 512 {
				break
			}
		}
		ex.waitMutex.Unlock()

		chosen, _, _ := reflect.Select(cases)
		if chosen == 0 {
			break
		}
		if chosen == 1 {
			time.Sleep(time.Millisecond * 50)
			continue
		}

		ex.Wakeup(sessions[chosen], ErrCancelContext)
	}
}

func (ex *Executer) changeWait() {
	if ex.waitCancel != nil {
		ex.waitCancel()
		ex.waitCancel = nil
	}
}

func (ex *Executer) pushWait(sessionID uint64, ctx context.Context) {
	ex.waitMutex.Lock()
	defer ex.waitMutex.Unlock()

	ex.waitSessions[sessionID] = ctx
	ex.changeWait()
}

func (ex *Executer) popWait(sessionID uint64) {
	ex.waitMutex.Lock()
	defer ex.waitMutex.Unlock()

	delete(ex.waitSessions, sessionID)
	ex.changeWait()
}

func (ex *Executer) work(initWg *sync.WaitGroup, workId int) {
	ex.cond.L.Lock()
	ex.OnWorkerCreate()
	initWg.Done()
	ex.cond.Wait()

	run := func(task Task, ctx context.Context) error {
		return task.Run(ctx)
	}

	ex.workId = workId
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
				ex.workId = workId
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
