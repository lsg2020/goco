package debug

import (
	"fmt"
	"sync/atomic"
)

func NewDebug(debugInfo string) (*Debug, error) {
	d := &Debug{debugInfo: debugInfo}
	return d, nil
}

type Debug struct {
	debugInfo string
	running   int64
	waiting   int64
}

func (d *Debug) String() string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("debug_info:%s running:%d waiting:%d",
		d.debugInfo,
		d.GetRunning(),
		d.GetWaiting(),
	)
}

func (d *Debug) GetRunning() int64 {
	if d == nil {
		return 0
	}
	return atomic.LoadInt64(&d.running)
}

func (d *Debug) AddRunning(v int64) {
	if d == nil {
		return
	}
	atomic.AddInt64(&d.running, v)
}

func (d *Debug) GetWaiting() int64 {
	if d == nil {
		return 0
	}
	return atomic.LoadInt64(&d.waiting)
}

func (d *Debug) AddWaiting(v int64) {
	if d == nil {
		return
	}
	atomic.AddInt64(&d.waiting, v)
}
