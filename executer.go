package co

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

func NewExecuter(ctx context.Context) (*Executer, error) {
	ex := &Executer{
		ctx: ctx,
	}
	err := ex.start(ctx, 5, 1024)
	if err != nil {
		return nil, err
	}
	return ex, nil
}

type taskInfo struct {
	ctx context.Context
	co  *Coroutine
	f   TaskFunc
	ch  chan error

	// wakeup
	isWakeup  bool
	sessionID uint64
	err       error
}

type Executer struct {
	ctx  context.Context
	cond *sync.Cond

	workId     int
	workAmount int
	waitAmount int

	tasks       chan *taskInfo
	nextSession uint64

	waitConds  map[uint64]*sync.Cond
	waitResult error

	waitMutex    sync.Mutex
	waitContext  context.Context
	waitCancel   context.CancelFunc
	waitSessions map[uint64]context.Context
}

func (ex *Executer) Run(ctx context.Context, f TaskFunc, co *Coroutine) error {
	if FromContextStatus(ctx) == StatusRunning {
		return ErrAlreadyInCoroutine
	}

	ch := make(chan error, 1)
	ex.tasks <- &taskInfo{ctx: ctx, co: co, f: f, ch: ch}
	err := <-ch
	return err
}

func (ex *Executer) wakeup(sessionID uint64, err error) {
	ex.tasks <- &taskInfo{isWakeup: true, sessionID: sessionID, err: err}
}

func (ex *Executer) PreWait() uint64 {
	sessionID := atomic.AddUint64(&ex.nextSession, 1)
	if sessionID == 0 {
		sessionID = atomic.AddUint64(&ex.nextSession, 1)
	}
	ex.waitConds[sessionID] = sync.NewCond(ex.cond.L)
	return sessionID
}

func (ex *Executer) Wait(ctx context.Context, sessionID uint64) error {
	if FromContextStatus(ctx) != StatusRunning {
		return ErrNeedFromCoroutine
	}

	waitCond, ok := ex.waitConds[sessionID]
	if !ok {
		return ErrWaitSessionMiss
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

	ex.pushWait(sessionID, ctx)

	ex.cond.Signal()
	waitCond.Wait()

	ex.popWait(sessionID)

	ex.workId = currentWorkId
	ex.waitAmount--
	delete(ex.waitConds, sessionID)

	return ex.waitResult
}

func (ex *Executer) start(ctx context.Context, initWorkAmount int, channelSize int) error {
	ex.cond = sync.NewCond(new(sync.Mutex))
	ex.nextSession = 0

	ex.ctx = ctx
	ex.tasks = make(chan *taskInfo, channelSize)

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
			if len(cases) > 8192 {
				break
			}
		}
		ex.waitMutex.Unlock()

		chosen, _, _ := reflect.Select(cases)
		if chosen == 0 {
			break
		}
		if chosen == 1 {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		ex.wakeup(sessions[chosen], ErrCancelContext)
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
	initWg.Done()
	ex.cond.Wait()

	ex.workId = workId
	for {
		select {
		case task := <-ex.tasks:
			if task.isWakeup {
				waitCond, ok := ex.waitConds[task.sessionID]
				if ok {
					ex.waitResult = task.err
					waitCond.Signal()
					ex.cond.Wait()
					ex.workId = workId
				}
				continue
			}

			ctx := WithContextCO(task.ctx, task.co)
			ctx = WithContextStatus(ctx, StatusRunning)
			func() {
				defer func() {
					if r := recover(); r != nil {
						task.ch <- fmt.Errorf("task panic, %v", r)
					}
				}()
				task.ch <- task.f(ctx)
			}()
		case <-ex.ctx.Done():
			return
		}
	}
}
