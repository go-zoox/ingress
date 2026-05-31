package service

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// DetailMetricsStreamer pushes route/service metrics over SSE.
type DetailMetricsStreamer struct {
	broker  *SSEBroker
	route   *RouteMetricsBuilder
	service *ServiceMetricsBuilder
	ingress *Ingress
	stop    chan struct{}
	seq     atomic.Int64

	routeState   sync.Map // *Subscriber -> map[string]any
	serviceState sync.Map

	throttleMu        sync.Mutex
	throttleScheduled bool
	lastPushAt        time.Time
}

// NewDetailMetricsStreamer creates an SSE publisher for detail metrics channels.
func NewDetailMetricsStreamer(
	broker *SSEBroker,
	ingress *Ingress,
	route *RouteMetricsBuilder,
	service *ServiceMetricsBuilder,
) *DetailMetricsStreamer {
	return &DetailMetricsStreamer{
		broker:  broker,
		ingress: ingress,
		route:   route,
		service: service,
		stop:    make(chan struct{}),
	}
}

// Start begins periodic pushes at the given interval.
func (s *DetailMetricsStreamer) Start(interval time.Duration) {
	if s == nil || s.broker == nil || interval <= 0 {
		return
	}
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
func (s *DetailMetricsStreamer) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

// ThrottledPushAll coalesces push requests.
func (s *DetailMetricsStreamer) ThrottledPushAll(minGap time.Duration) {
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

func (s *DetailMetricsStreamer) ForgetSubscriber(sub *Subscriber) {
	if s == nil || sub == nil {
		return
	}
	s.routeState.Delete(sub)
	s.serviceState.Delete(sub)
}

func (s *DetailMetricsStreamer) RouteSnapshotEvent(sub *Subscriber) SSEEvent {
	snap := s.routeSnapshot(sub)
	s.routeState.Store(sub, snap)
	return detailMetricsSSEEvent("route_metrics", "snapshot", snap)
}

func (s *DetailMetricsStreamer) ServiceSnapshotEvent(sub *Subscriber) SSEEvent {
	snap := s.serviceSnapshot(sub)
	s.serviceState.Store(sub, snap)
	return detailMetricsSSEEvent("service_metrics", "snapshot", snap)
}

func (s *DetailMetricsStreamer) pushAll() {
	s.broker.ForEachSubscriber("route_metrics", func(sub *Subscriber) {
		snap := s.routeSnapshot(sub)
		s.pushSubscriber(sub, "route_metrics", snap, &s.routeState)
	})
	s.broker.ForEachSubscriber("service_metrics", func(sub *Subscriber) {
		snap := s.serviceSnapshot(sub)
		s.pushSubscriber(sub, "service_metrics", snap, &s.serviceState)
	})
}

func (s *DetailMetricsStreamer) pushSubscriber(sub *Subscriber, channel string, snap map[string]any, state *sync.Map) {
	if s == nil || s.broker == nil || sub == nil || snap == nil {
		return
	}
	prevAny, hasPrev := state.Load(sub)
	prev, _ := prevAny.(map[string]any)
	if !hasPrev || prev == nil {
		state.Store(sub, snap)
		s.broker.SendJSONTo(sub, channel, "snapshot", snap)
		return
	}
	patch := computeMetricsSSEPatch(prev, snap)
	if patch.isEmpty() {
		return
	}
	patch.Seq = s.seq.Add(1)
	state.Store(sub, snap)
	s.broker.SendJSONTo(sub, channel, "patch", patch)
}

func (s *DetailMetricsStreamer) routeSnapshot(sub *Subscriber) map[string]any {
	if s == nil || s.route == nil || s.ingress == nil {
		return map[string]any{}
	}
	ri, pi, ok := subscriberRouteIndices(sub)
	if !ok {
		return map[string]any{}
	}
	cfg, err := s.ingress.LoadConfig()
	if err != nil {
		return map[string]any{}
	}
	rangeQ := metricsRangeForSubscriber(sub)
	window := WindowLabelForDuration(rangeQ.Duration())
	host, path, pathMatch := subscriberRouteScope(sub)
	analytics := s.route.Build(cfg, ri, pi, window, rangeQ, host, path, pathMatch)
	return RouteAnalyticsToMap(analytics)
}

func (s *DetailMetricsStreamer) serviceSnapshot(sub *Subscriber) map[string]any {
	if s == nil || s.service == nil || s.ingress == nil {
		return map[string]any{}
	}
	name := subscriberServiceName(sub)
	if name == "" {
		return map[string]any{}
	}
	content, err := s.ingress.ReadYAML()
	if err != nil {
		return map[string]any{}
	}
	catalog, err := ParseServiceCatalog(content)
	if err != nil {
		return map[string]any{}
	}
	entry, found := FindCatalogService(catalog, name)
	if !found {
		return map[string]any{}
	}
	cfg, err := s.ingress.LoadConfig()
	if err != nil {
		return map[string]any{}
	}
	refs := ListServiceRouteRefs(cfg, name)
	aliases := ServiceTargetAliases(entry, refs)
	rangeQ := metricsRangeForSubscriber(sub)
	window := WindowLabelForDuration(rangeQ.Duration())
	analytics := s.service.Build(window, rangeQ, aliases)
	return ServiceAnalyticsToMap(analytics)
}

func detailMetricsSSEEvent(channel, action string, payload any) SSEEvent {
	data, err := json.Marshal(payload)
	if err != nil {
		return SSEEvent{Event: channel + ":" + action, Data: "{}"}
	}
	return SSEEvent{
		Event: channel + ":" + action,
		Data:  string(data),
		ID:    time.Now().Format("20060102150405.000"),
	}
}
