package service

import (
	"sync"
	"time"
)

// LogStreamer tails access/error logs and publishes new lines on the SSE "logs" channel.
type LogStreamer struct {
	logs              *Logs
	broker            *SSEBroker
	onAccessLine      func()
	accessLineHandler func(line string)
	mu                sync.Mutex
	offsets           map[LogKind]int64
	stop              chan struct{}
}

// NewLogStreamer creates a log tail publisher.
func NewLogStreamer(logs *Logs, broker *SSEBroker) *LogStreamer {
	return &LogStreamer{
		logs:    logs,
		broker:  broker,
		offsets: map[LogKind]int64{
			LogAccess: 0,
			LogError:  0,
		},
		stop: make(chan struct{}),
	}
}

// SetOnAccessLine registers a callback when new access log lines are published.
func (s *LogStreamer) SetOnAccessLine(fn func()) {
	if s == nil {
		return
	}
	s.onAccessLine = fn
}

// SetAccessLineHandler parses each new access line (admin-only rollup when core hook is absent).
func (s *LogStreamer) SetAccessLineHandler(fn func(line string)) {
	if s == nil {
		return
	}
	s.accessLineHandler = fn
}

// Start begins polling log files at the given interval.
func (s *LogStreamer) Start(interval time.Duration) {
	if s == nil || s.logs == nil || s.broker == nil || interval <= 0 {
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
				s.poll()
			}
		}
	}()
}

// Stop ends background polling.
func (s *LogStreamer) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

func (s *LogStreamer) poll() {
	for _, kind := range []LogKind{LogAccess, LogError} {
		s.pollKind(kind)
	}
}

func (s *LogStreamer) pollKind(kind LogKind) {
	path := s.logs.logPath(kind)
	size, err := fileSize(path)
	if err != nil {
		return
	}

	s.mu.Lock()
	offset := s.offsets[kind]
	if offset == 0 && size > 0 {
		// Skip existing file content; only stream lines appended after start.
		s.offsets[kind] = size
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	result, err := s.logs.Search(LogQuery{Kind: kind, Offset: offset, Limit: 200})
	if err != nil {
		return
	}

	s.mu.Lock()
	s.offsets[kind] = result.Offset
	s.mu.Unlock()

	publishedAccess := false
	for _, line := range result.Lines {
		if line == "" {
			continue
		}
		s.broker.PublishJSON("logs", "line", map[string]string{
			"line": line,
			"kind": string(kind),
		})
		if kind == LogAccess {
			if s.accessLineHandler != nil {
				s.accessLineHandler(line)
			}
			publishedAccess = true
		}
	}
	if publishedAccess && s.onAccessLine != nil {
		s.onAccessLine()
	}
}
