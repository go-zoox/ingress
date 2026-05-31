package service

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// OverviewStreamer pushes aggregated overview snapshots to SSE subscribers.
type OverviewStreamer struct {
	builder  *OverviewBuilder
	broker   *SSEBroker
	interval time.Duration
	stop     chan struct{}
	seq      atomic.Int64
	subState sync.Map // *Subscriber -> OverviewSnapshot

	throttleMu        sync.Mutex
	throttleScheduled bool
	lastPushAt        time.Time
}

// NewOverviewStreamer creates an overview SSE publisher.
func NewOverviewStreamer(builder *OverviewBuilder, broker *SSEBroker) *OverviewStreamer {
	return &OverviewStreamer{
		builder: builder,
		broker:  broker,
		stop:    make(chan struct{}),
	}
}

// Start begins periodic snapshot pushes at the given interval.
func (s *OverviewStreamer) Start(interval time.Duration) {
	if s == nil || s.builder == nil || s.broker == nil || interval <= 0 {
		return
	}
	s.interval = interval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				s.pushAll()
			}
		}
	}()
}

// Stop ends background pushes.
func (s *OverviewStreamer) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

// Snapshot builds a snapshot for the given metrics window string.
func (s *OverviewStreamer) Snapshot(window string) OverviewSnapshot {
	if s == nil || s.builder == nil {
		return OverviewSnapshot{}
	}
	return s.builder.Snapshot(window)
}

// SnapshotEvent builds the initial full SSE snapshot for a subscriber.
func (s *OverviewStreamer) SnapshotEvent(sub *Subscriber) SSEEvent {
	snap := s.Snapshot(subParamWindow(sub))
	s.remember(sub, snap)
	return snapshotSSEEvent(snap)
}

// ForgetSubscriber clears cached overview state when a client disconnects.
func (s *OverviewStreamer) ForgetSubscriber(sub *Subscriber) {
	if s == nil || sub == nil {
		return
	}
	s.subState.Delete(sub)
}

// PushAll sends incremental updates to all overview subscribers.
func (s *OverviewStreamer) PushAll() {
	s.pushAll()
}

// ThrottledPushAll coalesces push requests to at most one push per minGap.
func (s *OverviewStreamer) ThrottledPushAll(minGap time.Duration) {
	if s == nil {
		return
	}
	if minGap <= 0 {
		s.pushAll()
		return
	}
	s.throttleMu.Lock()
	defer s.throttleMu.Unlock()
	now := time.Now()
	if now.Sub(s.lastPushAt) >= minGap {
		s.lastPushAt = now
		go s.pushAll()
		return
	}
	if s.throttleScheduled {
		return
	}
	s.throttleScheduled = true
	wait := minGap - now.Sub(s.lastPushAt)
	time.AfterFunc(wait, func() {
		s.throttleMu.Lock()
		s.throttleScheduled = false
		s.lastPushAt = time.Now()
		s.throttleMu.Unlock()
		s.pushAll()
	})
}

func (s *OverviewStreamer) pushAll() {
	cache := make(map[string]OverviewSnapshot)
	var mu sync.Mutex

	s.broker.ForEachSubscriber("overview", func(sub *Subscriber) {
		window := normalizeMetricsWindow(subParamWindow(sub))
		mu.Lock()
		snap, ok := cache[window]
		if !ok {
			snap = s.builder.Snapshot(window)
			cache[window] = snap
		}
		mu.Unlock()
		s.pushSubscriber(sub, snap, false)
	})
}

func (s *OverviewStreamer) pushSubscriber(sub *Subscriber, snap OverviewSnapshot, forceFull bool) {
	if s == nil || s.broker == nil || sub == nil {
		return
	}
	prev, hasPrev := s.load(sub)
	if forceFull || !hasPrev {
		s.remember(sub, snap)
		s.broker.SendJSONTo(sub, "overview", "snapshot", snap)
		return
	}
	patch := computeOverviewSSEPatch(prev, snap)
	if patch.isEmpty() {
		return
	}
	patch.Seq = s.seq.Add(1)
	s.remember(sub, snap)
	s.broker.SendJSONTo(sub, "overview", "patch", patch)
}

func (s *OverviewStreamer) remember(sub *Subscriber, snap OverviewSnapshot) {
	s.subState.Store(sub, snap)
}

func (s *OverviewStreamer) load(sub *Subscriber) (OverviewSnapshot, bool) {
	v, ok := s.subState.Load(sub)
	if !ok {
		return OverviewSnapshot{}, false
	}
	snap, ok := v.(OverviewSnapshot)
	return snap, ok
}

func subParamWindow(sub *Subscriber) string {
	if sub == nil {
		return ""
	}
	return sub.Param("window")
}

func snapshotSSEEvent(snap OverviewSnapshot) SSEEvent {
	data, err := json.Marshal(snap)
	if err != nil {
		return SSEEvent{Event: "overview:snapshot", Data: "{}"}
	}
	return SSEEvent{
		Event: "overview:snapshot",
		Data:  string(data),
		ID:    time.Now().Format("20060102150405.000"),
	}
}
