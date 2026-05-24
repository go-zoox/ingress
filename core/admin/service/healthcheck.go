package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	coresvc "github.com/go-zoox/ingress/core/service"
)

// HealthCheckResult stores the outcome of one health-check probe.
type HealthCheckResult struct {
	Key        string  `json:"key"`
	Host       string  `json:"host"`
	Path       string  `json:"path"`
	Backend    string  `json:"backend"`
	URL        string  `json:"url"`
	Status     string  `json:"status"` // "up", "down", "unknown"
	LastCheck  string  `json:"last_check"`
	ResponseMs float64 `json:"response_ms"`
	Error      string  `json:"error,omitempty"`
}

// HealthSummary is an aggregate of all health-check results.
type HealthSummary struct {
	Total   int `json:"total"`
	Up      int `json:"up"`
	Down    int `json:"down"`
	Unknown int `json:"unknown"`
}

// HealthCheckService periodically probes backends that have health checks configured.
type HealthCheckService struct {
	ingress  *Ingress
	broker   *SSEBroker
	client   *http.Client
	results  sync.Map // key -> *HealthCheckResult
	done     chan bool
	interval time.Duration
	timeout  time.Duration
}

// NewHealthCheckService creates a new health check service.
func NewHealthCheckService(ingress *Ingress, broker *SSEBroker) *HealthCheckService {
	return &HealthCheckService{
		ingress:  ingress,
		broker:   broker,
		interval: 30 * time.Second,
		timeout:  5 * time.Second,
		done:     make(chan bool, 1),
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Start launches the background health-check goroutine.
func (h *HealthCheckService) Start() {
	go h.run()
}

// Stop terminates the background goroutine.
func (h *HealthCheckService) Stop() {
	select {
	case h.done <- true:
	default:
	}
}

func (h *HealthCheckService) run() {
	// Perform an initial check immediately
	h.checkAll()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkAll()
		case <-h.done:
			return
		}
	}
}

func (h *HealthCheckService) checkAll() {
	cfg, err := h.ingress.LoadConfig()
	if err != nil {
		return
	}

	targets := h.extractTargets(cfg)
	for _, t := range targets {
		result := h.probe(t)
		// Check for status change
		prev, _ := h.results.Load(t.Key)
		statusChanged := prev == nil || prev.(*HealthCheckResult).Status != result.Status
		h.results.Store(t.Key, result)
		if statusChanged && h.broker != nil {
			h.broker.PublishJSON("health", "update", result)
		}
	}
}

// healthCheckTarget is a backend to probe.
type healthCheckTarget struct {
	Key            string
	Host           string
	Path           string
	Backend        string
	CheckPath      string
	Method         string
	ExpectedStatus []int
}

func (h *HealthCheckService) extractTargets(cfg *ingcore.Config) []healthCheckTarget {
	var targets []healthCheckTarget

	for i := range cfg.Rules {
		r := &cfg.Rules[i]
		// Rule-level backend
		hc := r.Backend.Service.HealthCheck
		if hc.Enable {
			target := h.buildTarget(r.Host, "/", r.Backend, hc)
			if target != nil {
				targets = append(targets, *target)
			}
		}
		// Path-level backends
		for j := range r.Paths {
			p := &r.Paths[j]
			hc := p.Backend.Service.HealthCheck
			if hc.Enable {
				target := h.buildTarget(r.Host, p.Path, p.Backend, hc)
				if target != nil {
					targets = append(targets, *target)
				}
			}
		}
	}

	return targets
}

func (h *HealthCheckService) buildTarget(host, path string, backend rule.Backend, hc coresvc.HealthCheck) *healthCheckTarget {
	target := backendTargetLabel(backend)
	if target == "" {
		return nil
	}

	checkPath := hc.Path
	if checkPath == "" {
		checkPath = "/health"
	}
	method := hc.Method
	if method == "" {
		method = "GET"
	}
	expectedStatus := hc.Status
	if len(expectedStatus) == 0 {
		expectedStatus = []int64{200}
	}

	var statusInts []int
	for _, s := range expectedStatus {
		statusInts = append(statusInts, int(s))
	}

	key := host + "|" + path + "|" + target

	return &healthCheckTarget{
		Key:            key,
		Host:           host,
		Path:           path,
		Backend:        target,
		CheckPath:      checkPath,
		Method:         method,
		ExpectedStatus: statusInts,
	}
}

func (h *HealthCheckService) probe(t healthCheckTarget) *HealthCheckResult {
	url := h.buildProbeURL(t.Backend, t.CheckPath)

	result := &HealthCheckResult{
		Key:     t.Key,
		Host:    t.Host,
		Path:    t.Path,
		Backend: t.Backend,
		URL:     url,
		Status:  "unknown",
	}

	if url == "" {
		return result
	}

	start := time.Now()
	req, err := http.NewRequest(t.Method, url, nil)
	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	req.Header.Set("Host", t.Host)

	resp, err := h.client.Do(req)
	elapsed := time.Since(start).Seconds() * 1000
	result.ResponseMs = float64(int(elapsed*100)) / 100
	result.LastCheck = time.Now().UTC().Format(time.RFC3339)

	if err != nil {
		result.Status = "down"
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	// Check if status matches expected
	matched := false
	for _, s := range t.ExpectedStatus {
		if resp.StatusCode == s {
			matched = true
			break
		}
	}

	if matched {
		result.Status = "up"
	} else {
		result.Status = "down"
		result.Error = fmt.Sprintf("unexpected status %d (expected %v)", resp.StatusCode, t.ExpectedStatus)
	}

	return result
}

// buildProbeURL constructs a probe URL from the backend target string.
func (h *HealthCheckService) buildProbeURL(backend, checkPath string) string {
	parts := strings.SplitN(backend, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	name := parts[0]
	port := parts[1]
	if name == "" {
		return ""
	}
	scheme := "http"
	if port == "443" {
		scheme = "https"
	}
	return scheme + "://" + name + ":" + port + checkPath
}

// backendTargetLabel returns a human-readable backend target string.
func backendTargetLabel(b rule.Backend) string {
	bt := backendTypeLabel(b)
	switch bt {
	case "redirect":
		if b.Redirect.URL != "" {
			return b.Redirect.URL
		}
		return "(redirect)"
	case "handler":
		return "(handler)"
	default:
		s := b.Service
		if s.Name == "" {
			return ""
		}
		port := s.Port
		if port == 0 {
			if s.Protocol == "https" {
				port = 443
			} else {
				port = 80
			}
		}
		return s.Name + ":" + strconv.FormatInt(port, 10)
	}
}

// backendTypeLabel determines the backend type string.
func backendTypeLabel(b rule.Backend) string {
	if b.Type != "" {
		return b.Type
	}
	if b.Redirect.URL != "" {
		return "redirect"
	}
	if b.Handler.Type != "" {
		return "handler"
	}
	return "service"
}

// ListResults returns all health-check results and a summary.
func (h *HealthCheckService) ListResults() ([]HealthCheckResult, HealthSummary) {
	var results []HealthCheckResult
	summary := HealthSummary{}

	h.results.Range(func(key, value interface{}) bool {
		r, ok := value.(*HealthCheckResult)
		if !ok {
			return true
		}
		results = append(results, *r)
		summary.Total++
		switch r.Status {
		case "up":
			summary.Up++
		case "down":
			summary.Down++
		default:
			summary.Unknown++
		}
		return true
	})

	if results == nil {
		results = []HealthCheckResult{}
	}

	return results, summary
}

// GetResult returns a single health check result by key.
func (h *HealthCheckService) GetResult(key string) (*HealthCheckResult, bool) {
	val, ok := h.results.Load(key)
	if !ok {
		return nil, false
	}
	r, ok := val.(*HealthCheckResult)
	return r, ok
}
