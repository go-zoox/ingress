package ratelimit

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-zoox/counter/bucket"
	rl "github.com/go-zoox/ratelimit"

	"github.com/go-zoox/ingress/core/rule"
)

const (
	KeyGlobal = "global"
	KeyRoute  = "route"
	KeyIP     = "ip"
	KeyHeader = "header"

	headerXRateLimitLimit           = "X-RateLimit-Limit"
	headerXRateLimitRemaining       = "X-RateLimit-Remaining"
	headerXRateLimitReset           = "X-RateLimit-Reset"
	headerXIngressRateLimit         = "X-Ingress-RateLimit"
	ingressRateLimitBlockHeaderVal  = "1"
)

// Policy is a compiled rate limit policy.
type Policy struct {
	rl       *rl.RateLimit
	Key      string
	Header   string
	TrustProxy bool
	XFFIndex int
}

// Ingress holds compiled global and per-rule limiters.
type Ingress struct {
	Global *Policy
	ByRule []*Policy // index aligned with cfg.Rules; nil when unset
}

// Compile builds rate limiters from config. Uses in-memory counters; when cache
// Redis is configured, limiters share the same Redis bucket settings.
func Compile(global rule.RateLimit, rules []rule.Rule, cacheHost string, cachePort int64, cacheUser, cachePass string, cacheDB int64, cachePrefix string) (*Ingress, error) {
	out := &Ingress{
		ByRule: make([]*Policy, len(rules)),
	}

	if enabled(global) {
		p, err := compilePolicy(global, "ingress:global", cacheHost, cachePort, cacheUser, cachePass, cacheDB, cachePrefix)
		if err != nil {
			return nil, fmt.Errorf("rate_limit: %w", err)
		}
		out.Global = p
	}

	for i := range rules {
		rlCfg := rules[i].RateLimit
		if !enabled(rlCfg) {
			continue
		}
		ns := fmt.Sprintf("ingress:rule:%d", i)
		p, err := compilePolicy(rlCfg, ns, cacheHost, cachePort, cacheUser, cachePass, cacheDB, cachePrefix)
		if err != nil {
			return nil, fmt.Errorf("rules[%d].rate_limit: %w", i, err)
		}
		out.ByRule[i] = p
	}

	return out, nil
}

func compilePolicy(cfg rule.RateLimit, namespace, cacheHost string, cachePort int64, cacheUser, cachePass string, cacheDB int64, cachePrefix string) (*Policy, error) {
	key := strings.ToLower(strings.TrimSpace(cfg.Key))
	if key == "" {
		key = KeyIP
	}
	switch key {
	case KeyGlobal, KeyRoute, KeyIP, KeyHeader:
	default:
		return nil, fmt.Errorf("unsupported key %q (use global, route, ip, or header)", cfg.Key)
	}
	if key == KeyHeader && strings.TrimSpace(cfg.Header) == "" {
		return nil, fmt.Errorf("header is required when key is header")
	}
	if cfg.Requests <= 0 {
		return nil, fmt.Errorf("requests must be positive")
	}
	if cfg.Period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	period := time.Duration(cfg.Period) * time.Second
	var limiter *rl.RateLimit
	var err error
	if strings.TrimSpace(cacheHost) != "" {
		prefix := cachePrefix
		if prefix == "" {
			prefix = "gozoox-ingress:"
		}
		limiter, err = rl.NewRedis(namespace, period, cfg.Requests, &bucket.RedisConfig{
			Host:     cacheHost,
			Port:     int(cachePort),
			Username: cacheUser,
			Password: cachePass,
			DB:       int(cacheDB),
			Prefix:   prefix,
		})
		if err != nil {
			return nil, err
		}
	} else {
		limiter = rl.NewMemory(namespace, period, cfg.Requests)
	}

	return &Policy{
		rl:         limiter,
		Key:        key,
		Header:     cfg.Header,
		TrustProxy: cfg.TrustProxy,
		XFFIndex:   cfg.XFFIndex,
	}, nil
}

func enabled(cfg rule.RateLimit) bool {
	if cfg.Enabled != nil {
		return *cfg.Enabled
	}
	return cfg.Requests > 0
}

// Result is the outcome of Check (after counters are incremented).
type Result struct {
	Blocked      bool
	Blocking     *Policy
	ScopeRuleIdx int // -1 when the blocking policy is global scope
}

// Check evaluates policies in order; returns blocked=true when any limit is exceeded.
func Check(req *http.Request, global *Policy, rulePolicy *Policy, ruleIdx int) Result {
	if global != nil {
		if blocked, _ := checkOne(req, global, ruleIdx, -1); blocked {
			return Result{Blocked: true, Blocking: global, ScopeRuleIdx: -1}
		}
	}
	if rulePolicy != nil {
		if blocked, _ := checkOne(req, rulePolicy, ruleIdx, ruleIdx); blocked {
			return Result{Blocked: true, Blocking: rulePolicy, ScopeRuleIdx: ruleIdx}
		}
	}
	return Result{}
}

func checkOne(req *http.Request, p *Policy, ruleIdx, scopeRuleIdx int) (blocked bool, retryAfterSec int64) {
	id := bucketID(req, p, ruleIdx, scopeRuleIdx)
	if err := p.rl.Inc(id); err != nil {
		// Fail open on counter errors.
		return false, 0
	}
	if !p.rl.IsExceeded(id) {
		return false, 0
	}
	resetMs := p.rl.ResetAfter(id)
	if resetMs < 0 {
		resetMs = 0
	}
	sec := resetMs / 1000
	if sec < 1 {
		sec = 1
	}
	return true, sec
}

func bucketID(req *http.Request, p *Policy, ruleIdx, scopeRuleIdx int) string {
	switch p.Key {
	case KeyGlobal:
		if scopeRuleIdx >= 0 {
			return fmt.Sprintf("global:rule:%d", scopeRuleIdx)
		}
		return "global"
	case KeyRoute:
		idx := ruleIdx
		if scopeRuleIdx >= 0 {
			idx = scopeRuleIdx
		}
		return fmt.Sprintf("route:%d", idx)
	case KeyIP:
		ip := ClientIP(req, p.TrustProxy, p.XFFIndex)
		idx := ruleIdx
		if scopeRuleIdx >= 0 {
			idx = scopeRuleIdx
		}
		if idx >= 0 {
			return fmt.Sprintf("route:%d:ip:%s", idx, ip)
		}
		return "ip:" + ip
	case KeyHeader:
		name := http.CanonicalHeaderKey(strings.TrimSpace(p.Header))
		val := req.Header.Get(name)
		if val == "" {
			val = "-"
		}
		if len(val) > 128 {
			val = val[:128]
		}
		idx := ruleIdx
		if scopeRuleIdx >= 0 {
			idx = scopeRuleIdx
		}
		if idx >= 0 {
			return fmt.Sprintf("route:%d:header:%s:%s", idx, name, val)
		}
		return fmt.Sprintf("header:%s:%s", name, val)
	default:
		return "unknown"
	}
}

// ClientIP returns the client IP, optionally from X-Forwarded-For.
func ClientIP(req *http.Request, trustProxy bool, xffIndex int) string {
	if req == nil {
		return "-"
	}
	if trustProxy {
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			idx := xffIndex
			if idx < 0 {
				idx = len(parts) + idx
			}
			if idx >= 0 && idx < len(parts) && parts[idx] != "" {
				return parts[idx]
			}
		}
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if req.RemoteAddr != "" {
		return req.RemoteAddr
	}
	return "-"
}

// QuotaFor returns limit, remaining, and reset time (Unix seconds) for a policy bucket (after Inc).
func QuotaFor(req *http.Request, p *Policy, ruleIdx, scopeRuleIdx int) (limit, remaining, resetUnixSec int64) {
	if p == nil {
		return 0, 0, 0
	}
	id := bucketID(req, p, ruleIdx, scopeRuleIdx)
	st, err := p.rl.Status(id)
	if err != nil {
		limit = p.rl.Total(id)
		remaining = p.rl.Remaining(id)
	} else {
		limit = st.Total
		remaining = st.Remaining
	}
	if remaining < 0 {
		remaining = 0
	}
	resetMs := p.rl.ResetAt(id)
	if resetMs > 0 {
		resetUnixSec = resetMs / 1000
	}
	return limit, remaining, resetUnixSec
}

// WriteResponseHeaders sets X-RateLimit-* and, when blocked, X-Ingress-RateLimit.
// On success, reports the per-rule policy when present, otherwise global.
func WriteResponseHeaders(setHeader func(key, value string), req *http.Request, global, rulePolicy *Policy, ruleIdx int, res Result) {
	if setHeader == nil {
		return
	}
	var p *Policy
	scope := -1
	if res.Blocked && res.Blocking != nil {
		setHeader(headerXIngressRateLimit, ingressRateLimitBlockHeaderVal)
		p = res.Blocking
		scope = res.ScopeRuleIdx
	} else if rulePolicy != nil {
		p = rulePolicy
		scope = ruleIdx
	} else if global != nil {
		p = global
		scope = -1
	} else {
		return
	}
	limit, remaining, reset := QuotaFor(req, p, ruleIdx, scope)
	if res.Blocked {
		remaining = 0
	}
	setHeader(headerXRateLimitLimit, strconv.FormatInt(limit, 10))
	setHeader(headerXRateLimitRemaining, strconv.FormatInt(remaining, 10))
	if reset > 0 {
		setHeader(headerXRateLimitReset, strconv.FormatInt(reset, 10))
	}
}
