package service

import (
	"testing"
	"time"
)

func TestRequest_Delay(t *testing.T) {
	tests := []struct {
		name     string
		delay    int64
		expected time.Duration
	}{
		{
			name:     "zero delay",
			delay:    0,
			expected: 0,
		},
		{
			name:     "100ms delay",
			delay:    100,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "1000ms delay",
			delay:    1000,
			expected: 1000 * time.Millisecond,
		},
		{
			name:     "500ms delay",
			delay:    500,
			expected: 500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{
				Delay: tt.delay,
			}

			if req.Delay != tt.delay {
				t.Errorf("expected delay %d, got %d", tt.delay, req.Delay)
			}

			// Verify the duration conversion
			if tt.delay > 0 {
				delayDuration := time.Duration(req.Delay) * time.Millisecond
				if delayDuration != tt.expected {
					t.Errorf("expected duration %v, got %v", tt.expected, delayDuration)
				}
			}
		})
	}
}

func TestRequest_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  int64
		expected time.Duration
	}{
		{
			name:     "zero timeout",
			timeout:  0,
			expected: 0,
		},
		{
			name:     "5 seconds timeout",
			timeout:  5,
			expected: 5 * time.Second,
		},
		{
			name:     "30 seconds timeout",
			timeout:  30,
			expected: 30 * time.Second,
		},
		{
			name:     "60 seconds timeout",
			timeout:  60,
			expected: 60 * time.Second,
		},
		{
			name:     "1 second timeout",
			timeout:  1,
			expected: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{
				Timeout: tt.timeout,
			}

			if req.Timeout != tt.timeout {
				t.Errorf("expected timeout %d, got %d", tt.timeout, req.Timeout)
			}

			// Verify the duration conversion
			if tt.timeout > 0 {
				timeoutDuration := time.Duration(req.Timeout) * time.Second
				if timeoutDuration != tt.expected {
					t.Errorf("expected duration %v, got %v", tt.expected, timeoutDuration)
				}
			}
		})
	}
}

func TestRequest_DelayAndTimeout(t *testing.T) {
	req := Request{
		Delay:   100,
		Timeout: 30,
	}

	if req.Delay != 100 {
		t.Errorf("expected delay 100, got %d", req.Delay)
	}

	if req.Timeout != 30 {
		t.Errorf("expected timeout 30, got %d", req.Timeout)
	}

	// Verify both can be set together
	delayDuration := time.Duration(req.Delay) * time.Millisecond
	timeoutDuration := time.Duration(req.Timeout) * time.Second

	if delayDuration != 100*time.Millisecond {
		t.Errorf("expected delay duration 100ms, got %v", delayDuration)
	}

	if timeoutDuration != 30*time.Second {
		t.Errorf("expected timeout duration 30s, got %v", timeoutDuration)
	}
}

func TestService_RequestDelayAndTimeout(t *testing.T) {
	s := &Service{
		Name: "test-service",
		Port: 8080,
		Request: Request{
			Delay:   200,
			Timeout: 10,
		},
	}

	if s.Request.Delay != 200 {
		t.Errorf("expected delay 200, got %d", s.Request.Delay)
	}

	if s.Request.Timeout != 10 {
		t.Errorf("expected timeout 10, got %d", s.Request.Timeout)
	}
}
