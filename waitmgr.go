package co

import (
	"container/list"
	"context"
	"reflect"
	"sync"
	"time"
)

func NewWaitSessionMgr(ctx context.Context, shards uint64, wakeup wakeupCB) *WaitSessionMgr {
	mgr := &WaitSessionMgr{
		shards:  shards,
		storage: make([]*waitShardNode, shards),
		wakeup:  wakeup,
	}
	for i := uint64(0); i < shards; i++ {
		info := &waitShardNode{
			sessionMap:   make(map[uint64]*list.Element),
			sessionMapEx: make(map[uint64]*waitSessionInfo),
		}
		mgr.storage[i] = info
		go mgr.waitMonitor(ctx, info)
	}

	return mgr
}

type waitSessionInfo struct {
	sessionID   uint64
	ctx         context.Context
	haveTimeout bool
}

type wakeupCB func(sessionID uint64)

type waitShardNode struct {
	mutex        sync.RWMutex
	sessionList  list.List
	sessionMap   map[uint64]*list.Element
	sessionMapEx map[uint64]*waitSessionInfo
	cancel       context.CancelFunc
}

type WaitSessionMgr struct {
	shards  uint64
	storage []*waitShardNode
	wakeup  wakeupCB
}

func (mgr *WaitSessionMgr) Size() (int, int) {
	size := 0
	sizeEx := 0
	for i := uint64(0); i < mgr.shards; i++ {
		mgr.storage[i].mutex.RLock()
		size += len(mgr.storage[i].sessionMap)
		sizeEx += len(mgr.storage[i].sessionMapEx)
		mgr.storage[i].mutex.RUnlock()
	}
	return size, sizeEx
}

func (mgr *WaitSessionMgr) PushWait(sessionID uint64, ctx context.Context) {
	_, haveTimeout := ctx.Deadline()

	shardInfo := mgr.storage[sessionID%mgr.shards]

	shardInfo.mutex.Lock()
	defer shardInfo.mutex.Unlock()

	v := &waitSessionInfo{sessionID: sessionID, ctx: ctx, haveTimeout: haveTimeout}
	if !haveTimeout {
		shardInfo.sessionMapEx[sessionID] = v
	} else {
		e := shardInfo.sessionList.PushBack(v)
		shardInfo.sessionMap[sessionID] = e
	}
	mgr.changeWait(shardInfo)
}

func (mgr *WaitSessionMgr) PopWait(sessionID uint64) bool {
	shardInfo := mgr.storage[sessionID%mgr.shards]

	shardInfo.mutex.Lock()
	defer shardInfo.mutex.Unlock()

	if e, ok := shardInfo.sessionMap[sessionID]; ok {
		delete(shardInfo.sessionMap, sessionID)
		shardInfo.sessionList.Remove(e)
		mgr.changeWait(shardInfo)
		return true
	} else if _, ok := shardInfo.sessionMapEx[sessionID]; ok {
		delete(shardInfo.sessionMapEx, sessionID)
		mgr.changeWait(shardInfo)
		return true
	}
	return false
}

func (mgr *WaitSessionMgr) changeWait(shardInfo *waitShardNode) {
	if shardInfo.cancel != nil {
		shardInfo.cancel()
		shardInfo.cancel = nil
	}
}

func (mgr *WaitSessionMgr) waitMonitor(root context.Context, info *waitShardNode) {
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
		var ctx context.Context

		info.mutex.RLock()
		mgr.changeWait(info)
		ctx, info.cancel = context.WithCancel(root)
		cases, sessions = pushCase(cases, sessions, 0, root) // wait exit
		cases, sessions = pushCase(cases, sessions, 0, ctx)  // wait change
		for e := info.sessionList.Front(); e != nil; e = e.Next() {
			wsi, ok := e.Value.(*waitSessionInfo)
			if !ok {
				continue
			}
			cases, sessions = pushCase(cases, sessions, wsi.sessionID, wsi.ctx) // nolint
			if len(cases) >= 128 {
				break
			}
		}
		if len(info.sessionMapEx) > 0 {
			amount := 0
			for _, v := range info.sessionMapEx {
				cases, sessions = pushCase(cases, sessions, v.sessionID, v.ctx) // nolint
				amount++
				if amount >= 128 {
					break
				}
			}
		}
		info.mutex.RUnlock()

		chosen, _, _ := reflect.Select(cases)
		if chosen == 0 {
			break
		} else if chosen == 1 {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		sessionID := sessions[chosen]
		mgr.wakeup(sessionID)
	}
}
