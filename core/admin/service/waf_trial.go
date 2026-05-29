package service

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/waf"
)

// WAFTrialInput is the request body for WAF dry-run matching.
type WAFTrialInput struct {
	Host         string            `json:"host"`
	Path         string            `json:"path"`
	Method       string            `json:"method"`
	ClientIP     string            `json:"client_ip"`
	Query        string            `json:"query"`
	Headers      map[string]string `json:"headers"`
	RuleIndex    *int              `json:"rule_index"`
	EventID      *uint             `json:"event_id"`
	ExpectedRule string            `json:"expected_rule"`
}

// WAFTrialHit is one WAF evaluation hit.
type WAFTrialHit struct {
	Action   string `json:"action"`
	Rule     string `json:"rule"`
	ClientIP string `json:"client_ip"`
}

// WAFTrialResult is the WAF dry-run response.
type WAFTrialResult struct {
	Matched            bool          `json:"matched"`
	WouldBlock         bool          `json:"would_block"`
	RuleIndex          int           `json:"rule_index"`
	PathIndex          int           `json:"path_index"`
	Host               string        `json:"host"`
	Path               string        `json:"path"`
	WAFEnabled         bool          `json:"waf_enabled"`
	ConfigWAFEnabled   bool          `json:"config_waf_enabled"`
	RuntimeWAFEnabled  bool          `json:"runtime_waf_enabled"`
	LogOnly            bool          `json:"log_only"`
	Hits               []WAFTrialHit `json:"hits"`
	ExpectedRule       string        `json:"expected_rule,omitempty"`
	ExpectedRuleHit    bool          `json:"expected_rule_hit"`
	Message            string        `json:"message,omitempty"`
	Hint               string        `json:"hint,omitempty"`
}

// TrialWAF evaluates a synthetic request against compiled WAF profiles.
func (ing *Ingress) TrialWAF(in WAFTrialInput) (WAFTrialResult, error) {
	host := strings.TrimSpace(in.Host)
	path := strings.TrimSpace(in.Path)
	if host == "" {
		return WAFTrialResult{}, fmt.Errorf("host is required")
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	method := strings.TrimSpace(in.Method)
	if method == "" {
		method = http.MethodGet
	}

	cfg, err := ing.LoadConfig()
	if err != nil {
		return WAFTrialResult{}, err
	}
	if err := ingcore.PrepareForInspect(cfg); err != nil {
		return WAFTrialResult{}, err
	}

	configWAFOn := cfg.WAF.Enabled
	runtimeWAFOn := ing.runtimeWAFEnabled(cfg)
	effectiveWAFOn := configWAFOn || runtimeWAFOn

	expectedRule := strings.TrimSpace(in.ExpectedRule)
	if in.EventID != nil && *in.EventID > 0 {
		if ev, err := NewAudit().GetWAFEvent(*in.EventID); err == nil && ev != nil {
			if expectedRule == "" {
				expectedRule = strings.TrimSpace(ev.Rule)
			}
		}
	}

	globalWAF := cfg.WAF
	if effectiveWAFOn && !globalWAF.Enabled {
		globalWAF.Enabled = true
	}

	perRule, fallback, err := waf.CompileIngress(globalWAF, cfg.Rules)
	if err != nil {
		return WAFTrialResult{}, err
	}

	ruleIndex := -1
	pathIndex := -1
	if in.RuleIndex != nil {
		ruleIndex = *in.RuleIndex
	} else {
		preview, err := ingcore.PreviewMatch(cfg, host, path)
		if err != nil {
			return WAFTrialResult{}, err
		}
		if preview.Matched {
			ruleIndex = preview.RuleIndex
			pathIndex = preview.PathIndex
		}
	}

	var prof *waf.Profile
	switch {
	case ruleIndex >= 0 && ruleIndex < len(perRule):
		prof = perRule[ruleIndex]
	case fallback != nil:
		prof = fallback
		ruleIndex = -1
	default:
		return WAFTrialResult{
			Matched:    false,
			Host:       host,
			Path:       path,
			WAFEnabled: cfg.WAF.Enabled,
			Message:    "no WAF profile for host/path",
		}, nil
	}

	if !effectiveWAFOn {
		return WAFTrialResult{
			Matched:           false,
			Host:              host,
			Path:              path,
			RuleIndex:         ruleIndex,
			PathIndex:         pathIndex,
			WAFEnabled:        false,
			ConfigWAFEnabled:  configWAFOn,
			RuntimeWAFEnabled: runtimeWAFOn,
			ExpectedRule:      expectedRule,
			Message:           "WAF 未启用",
			Hint:              "请在配置中设置 waf.enabled: true，或在 WAF 页打开运行时开关后再试匹配",
		}, nil
	}

	if prof == nil || !prof.Enabled {
		return WAFTrialResult{
			Matched:           false,
			Host:              host,
			Path:              path,
			RuleIndex:         ruleIndex,
			PathIndex:         pathIndex,
			WAFEnabled:        false,
			ConfigWAFEnabled:  configWAFOn,
			RuntimeWAFEnabled: runtimeWAFOn,
			ExpectedRule:      expectedRule,
			Message:           "WAF disabled for matched scope",
			Hint:              "匹配到的路由范围内 WAF 配置为关闭；检查该 host 对应 rules[].waf 或全局 waf.enabled",
		}, nil
	}

	if waf.HostSkipsWAF(prof, host) {
		return WAFTrialResult{
			Matched:           false,
			WouldBlock:        false,
			RuleIndex:         ruleIndex,
			PathIndex:         pathIndex,
			Host:              host,
			Path:              path,
			WAFEnabled:        true,
			ConfigWAFEnabled:  configWAFOn,
			RuntimeWAFEnabled: runtimeWAFOn,
			LogOnly:           prof.GlobalLogOnly,
			ExpectedRule:      expectedRule,
			Message:           "WAF skipped for host (allow_hosts)",
			Hint:              "该 Host 在 waf.allow_hosts 域名白名单中，跳过全部 WAF 检查",
		}, nil
	}

	rawURL := "http://" + host + path
	if q := strings.TrimSpace(in.Query); q != "" {
		if strings.HasPrefix(q, "?") {
			rawURL += q
		} else {
			rawURL += "?" + q
		}
	}
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return WAFTrialResult{}, err
	}
	for k, v := range in.Headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}
	if ip := strings.TrimSpace(in.ClientIP); ip != "" {
		if hostIP, _, splitErr := net.SplitHostPort(ip); splitErr == nil {
			req.RemoteAddr = hostIP + ":0"
		} else {
			req.RemoteAddr = ip + ":0"
		}
	}

	var hits []WAFTrialHit
	reportFn := func(action, rule, cliIP string) {
		hits = append(hits, WAFTrialHit{Action: action, Rule: rule, ClientIP: cliIP})
	}

	pathOnly := path
	if u, parseErr := url.Parse(rawURL); parseErr == nil && u.Path != "" {
		pathOnly = u.Path
	}

	wouldBlock := waf.CheckRequest(prof, req, host, pathOnly, method, reportFn)

	expectedHit := false
	if expectedRule != "" {
		for _, h := range hits {
			if ruleHitMatchesExpected(h.Rule, expectedRule) {
				expectedHit = true
				break
			}
		}
	}

	out := WAFTrialResult{
		Matched:           len(hits) > 0,
		WouldBlock:        wouldBlock,
		RuleIndex:         ruleIndex,
		PathIndex:         pathIndex,
		Host:              host,
		Path:              path,
		WAFEnabled:        true,
		ConfigWAFEnabled:  configWAFOn,
		RuntimeWAFEnabled: runtimeWAFOn,
		LogOnly:           prof.GlobalLogOnly,
		Hits:              hits,
		ExpectedRule:      expectedRule,
		ExpectedRuleHit:   expectedHit,
	}
	if expectedRule != "" && !expectedHit {
		if rd := LookupWAFRule(cfg, expectedRule); rd != nil && rd.Source == "demo" {
			out.Hint = "列表中的事件可能来自演示种子数据，当前 ingress 配置未包含该规则；可在配置页添加对应 waf.rules 或忽略演示事件"
		} else if len(hits) == 0 {
			out.Hint = "未命中任何规则：检查 User-Agent/路径是否与事件当时一致，以及 waf.log_only、disable_builtin 等配置"
		} else {
			out.Hint = "已命中 WAF，但规则名与事件记录不一致（实际命中见下方列表）"
		}
	}
	return out, nil
}

func (ing *Ingress) runtimeWAFEnabled(cfg *ingcore.Config) bool {
	if cfg.WAF.Enabled {
		return true
	}
	if ing.cfg != nil && ing.cfg.CoreInstance != nil {
		return ing.cfg.CoreInstance.IsWAFEnabled()
	}
	return false
}

func ruleHitMatchesExpected(hitRule, expected string) bool {
	hitRule = strings.TrimSpace(strings.ToLower(hitRule))
	expected = strings.TrimSpace(strings.ToLower(expected))
	if hitRule == "" || expected == "" {
		return false
	}
	if hitRule == expected || strings.Contains(hitRule, expected) || strings.Contains(expected, hitRule) {
		return true
	}
	if strings.HasPrefix(hitRule, "sig ") {
		id := strings.TrimSpace(strings.TrimPrefix(hitRule, "sig "))
		if id == expected || strings.Contains(expected, id) {
			return true
		}
	}
	return false
}
