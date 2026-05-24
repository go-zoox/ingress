package service

import (
	"encoding/json"
	"sync"
	"time"
)

// SSEEvent is a single Server-Sent Event payload.
type SSEEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
	Retry int    `json:"retry,omitempty"`
}

// Subscriber represents a connected SSE client.
type Subscriber struct {
	Ch       chan SSEEvent
	Channels []string
	Closed   bool
}

// SSEBroker manages channel-based pub/sub for SSE clients.
type SSEBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[*Subscriber]struct{} // channel -> set of subscribers
	ipCount     map[string]int                      // ip -> connection count
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		subscribers: make(map[string]map[*Subscriber]struct{}),
		ipCount:     make(map[string]int),
	}
}

// Subscribe creates a new subscriber for the given channels.
// Returns the subscriber or an error if the IP has too many connections.
func (b *SSEBroker) Subscribe(channels []string, clientIP string) (*Subscriber, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Enforce per-IP connection limit
	const maxPerIP = 5
	if clientIP != "" && b.ipCount[clientIP] >= maxPerIP {
		return nil, ErrTooManyConnections
	}

	ch := make(chan SSEEvent, 32)
	sub := &Subscriber{
		Ch:       ch,
		Channels: channels,
		Closed:   false,
	}

	for _, c := range channels {
		if b.subscribers[c] == nil {
			b.subscribers[c] = make(map[*Subscriber]struct{})
		}
		b.subscribers[c][sub] = struct{}{}
	}

	if clientIP != "" {
		b.ipCount[clientIP]++
	}

	return sub, nil
}

// Unsubscribe removes a subscriber from all its channels.
func (b *SSEBroker) Unsubscribe(sub *Subscriber, clientIP string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub.Closed {
		return
	}
	sub.Closed = true

	for _, c := range sub.Channels {
		if subs, ok := b.subscribers[c]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(b.subscribers, c)
			}
		}
	}

	close(sub.Ch)

	if clientIP != "" {
		if b.ipCount[clientIP] > 0 {
			b.ipCount[clientIP]--
			if b.ipCount[clientIP] == 0 {
				delete(b.ipCount, clientIP)
			}
		}
	}
}

// Publish sends an event to all subscribers of the given channel.
func (b *SSEBroker) Publish(channel string, event SSEEvent) {
	b.mu.RLock()
	subs, ok := b.subscribers[channel]
	b.mu.RUnlock()

	if !ok {
		return
	}

	// Non-blocking send to each subscriber
	for sub := range subs {
		select {
		case sub.Ch <- event:
		default:
			// Drop event if subscriber channel is full
		}
	}
}

// PublishJSON marshals v and publishes it as a data field on the channel.
func (b *SSEBroker) PublishJSON(channel, action string, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	b.Publish(channel, SSEEvent{
		Event: channel + ":" + action,
		Data:  string(data),
		ID:    time.Now().Format("20060102150405.000"),
	})
}

var ErrTooManyConnections = &tooManyConnectionsError{}

type tooManyConnectionsError struct{}

func (e *tooManyConnectionsError) Error() string {
	return "too many SSE connections from this IP (max 5)"
}
