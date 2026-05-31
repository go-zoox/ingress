package service

import (
	"sync/atomic"

	ingcore "github.com/go-zoox/ingress/core"
)

const defaultRollupQueueSize = 8192

// AsyncRollupRecorder enqueues access metrics off the ingress request path.
type AsyncRollupRecorder struct {
	rollup  *MetricsRollup
	ch      chan AccessEntry
	dropped atomic.Uint64
}

// NewAsyncRollupRecorder starts a background worker that calls MetricsRollup.Record.
func NewAsyncRollupRecorder(rollup *MetricsRollup, queueSize int) *AsyncRollupRecorder {
	if queueSize <= 0 {
		queueSize = defaultRollupQueueSize
	}
	r := &AsyncRollupRecorder{
		rollup: rollup,
		ch:     make(chan AccessEntry, queueSize),
	}
	go r.run()
	return r
}

func (r *AsyncRollupRecorder) run() {
	if r == nil {
		return
	}
	for e := range r.ch {
		if r.rollup != nil {
			r.rollup.Record(e)
		}
	}
}

// Enqueue schedules one event; drops when the queue is full.
func (r *AsyncRollupRecorder) Enqueue(ev ingcore.AccessMetricsEvent) {
	if r == nil || r.ch == nil {
		return
	}
	e := AccessEntryFromCoreEvent(ev)
	select {
	case r.ch <- e:
	default:
		r.dropped.Add(1)
	}
}

// Dropped returns the number of events dropped due to a full queue.
func (r *AsyncRollupRecorder) Dropped() uint64 {
	if r == nil {
		return 0
	}
	return r.dropped.Load()
}
