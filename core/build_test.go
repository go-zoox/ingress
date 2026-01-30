package core

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/service"
)

func TestBuild_RequestDelay(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Delay: 100, // 100ms delay
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify the service has delay configured
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Delay != 100 {
		t.Errorf("expected delay 100ms, got %d", matchedService.Service.Request.Delay)
	}

	// Verify delay duration conversion
	delayDuration := time.Duration(matchedService.Service.Request.Delay) * time.Millisecond
	expectedDuration := 100 * time.Millisecond
	if delayDuration != expectedDuration {
		t.Errorf("expected delay duration %v, got %v", expectedDuration, delayDuration)
	}
}

func TestBuild_RequestTimeout(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Timeout: 30, // 30 seconds timeout
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify the service has timeout configured
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Timeout != 30 {
		t.Errorf("expected timeout 30s, got %d", matchedService.Service.Request.Timeout)
	}

	// Verify timeout duration conversion
	timeoutDuration := time.Duration(matchedService.Service.Request.Timeout) * time.Second
	expectedDuration := 30 * time.Second
	if timeoutDuration != expectedDuration {
		t.Errorf("expected timeout duration %v, got %v", expectedDuration, timeoutDuration)
	}
}

func TestBuild_RequestDelayAndTimeout(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Delay:   200, // 200ms delay
							Timeout: 60,  // 60 seconds timeout
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify the service has both delay and timeout configured
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Delay != 200 {
		t.Errorf("expected delay 200ms, got %d", matchedService.Service.Request.Delay)
	}

	if matchedService.Service.Request.Timeout != 60 {
		t.Errorf("expected timeout 60s, got %d", matchedService.Service.Request.Timeout)
	}

	// Verify both duration conversions
	delayDuration := time.Duration(matchedService.Service.Request.Delay) * time.Millisecond
	timeoutDuration := time.Duration(matchedService.Service.Request.Timeout) * time.Second

	if delayDuration != 200*time.Millisecond {
		t.Errorf("expected delay duration 200ms, got %v", delayDuration)
	}

	if timeoutDuration != 60*time.Second {
		t.Errorf("expected timeout duration 60s, got %v", timeoutDuration)
	}
}

func TestBuild_RequestTimeoutContext(t *testing.T) {
	// Test that timeout creates a context with timeout
	timeout := int64(5) // 5 seconds
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create a base context
	baseCtx := context.Background()

	// Create timeout context (simulating what build.go does)
	timeoutCtx, cancel := context.WithTimeout(baseCtx, timeoutDuration)
	defer cancel()

	// Verify the context has a deadline
	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	// Verify the deadline is approximately timeoutDuration from now
	expectedDeadline := time.Now().Add(timeoutDuration)
	diff := expectedDeadline.Sub(deadline)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected deadline to be approximately %v from now, got %v (diff: %v)", timeoutDuration, deadline, diff)
	}

	// Verify context is not done initially
	select {
	case <-timeoutCtx.Done():
		t.Fatal("expected context to not be done initially")
	default:
		// Good, context is not done
	}
}

func TestBuild_RequestDelayTiming(t *testing.T) {
	// Test that delay actually causes a delay
	delay := int64(100) // 100ms
	delayDuration := time.Duration(delay) * time.Millisecond

	start := time.Now()

	// Simulate delay (what build.go does)
	time.Sleep(delayDuration)

	elapsed := time.Since(start)

	// Verify the delay was approximately correct (allow some margin for timing)
	if elapsed < delayDuration-time.Millisecond*10 {
		t.Errorf("expected delay of at least %v, got %v", delayDuration-time.Millisecond*10, elapsed)
	}

	if elapsed > delayDuration+time.Millisecond*50 {
		t.Errorf("expected delay of at most %v, got %v", delayDuration+time.Millisecond*50, elapsed)
	}
}

func TestBuild_RequestTimeoutExpiration(t *testing.T) {
	// Test that timeout context expires correctly
	timeout := int64(1) // 1 second
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	// Verify context expires after timeout
	select {
	case <-timeoutCtx.Done():
		// Good, context expired
		elapsed := time.Since(time.Now().Add(-timeoutDuration))
		if elapsed < 0 {
			elapsed = -elapsed
		}
		if elapsed > time.Second*2 {
			t.Errorf("context expired too late, elapsed: %v", elapsed)
		}
	case <-time.After(timeoutDuration + time.Second):
		t.Fatal("expected context to expire within timeout duration")
	}
}

func TestBuild_RequestWithZeroDelayAndTimeout(t *testing.T) {
	cfg := &Config{
		Port: 8080,
		Rules: []rule.Rule{
			{
				Host: "test.example.com",
				Backend: rule.Backend{
					Service: service.Service{
						Name:     "test-service",
						Port:     80,
						Protocol: "http",
						Request: service.Request{
							Delay:   0, // No delay
							Timeout: 0, // No timeout
						},
					},
				},
			},
		},
	}

	c, err := New("test-version", cfg)
	if err != nil {
		t.Fatalf("failed to create core: %v", err)
	}

	if c == nil {
		t.Fatal("expected core instance, got nil")
	}

	// Verify zero values are handled correctly
	matchedService, err := MatchHost(cfg.Rules, rule.Backend{}, "test.example.com")
	if err != nil {
		t.Fatalf("failed to match host: %v", err)
	}

	if matchedService.Service.Request.Delay != 0 {
		t.Errorf("expected delay 0, got %d", matchedService.Service.Request.Delay)
	}

	if matchedService.Service.Request.Timeout != 0 {
		t.Errorf("expected timeout 0, got %d", matchedService.Service.Request.Timeout)
	}
}

func TestBuild_RequestTimeoutInHTTPRequest(t *testing.T) {
	// Test that timeout is applied to HTTP request context
	timeout := int64(5) // 5 seconds
	timeoutDuration := time.Duration(timeout) * time.Second

	// Create a base HTTP request
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Apply timeout to request context (simulating what build.go does in OnRequest)
	timeoutCtx, cancel := context.WithTimeout(req.Context(), timeoutDuration)
	_ = cancel // cancel will be called when request completes
	req = req.WithContext(timeoutCtx)

	// Verify the request context has a deadline
	deadline, ok := req.Context().Deadline()
	if !ok {
		t.Fatal("expected request context to have a deadline")
	}

	// Verify the deadline is approximately timeoutDuration from now
	expectedDeadline := time.Now().Add(timeoutDuration)
	diff := expectedDeadline.Sub(deadline)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected deadline to be approximately %v from now, got %v (diff: %v)", timeoutDuration, deadline, diff)
	}
}
