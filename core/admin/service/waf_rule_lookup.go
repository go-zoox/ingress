package service

import (
	"strings"
	"time"

	"github.com/go-zoox/ingress/core/admin/model"
	ingcore "github.com/go-zoox/ingress/core"
	"github.com/go-zoox/ingress/core/rule"
	"github.com/go-zoox/ingress/core/waf"
)

// WAFRuleDetail describes a WAF hit rule for the admin UI.
type WAFRuleDetail struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Phase       string   `json:"phase"`
	Type        string   `json:"type"`
	Pattern     string   `json:"pattern,omitempty"`
	Targets     []string `json:"targets,omitempty"`
	Source      string   `json:"source"` // config | builtin | phase | demo
	Description string   `json:"description"`
	LogOnly     bool     `json:"log_only,omitempty"`
	Enabled     bool     `json:"enabled"`
	Builtin     bool     `json:"builtin,omitempty"`
}

// WAFEventDetail is a WAF event plus resolved rule metadata.
type WAFEventDetail struct {
	ID        uint   `json:"id"`
	Action    string `json:"action"`
	Rule      string `json:"rule"`
	Host      string `json:"host"`
	Path      string `json:"path"`
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
	RuleDetail *WAFRuleDetail `json:"rule_detail"`
	ReplayNote string         `json:"replay_note"`
}

// demoRuleCatalog mirrors seeded waf_events in bootstrap/sample.go (not in ingress.yaml).
var demoRuleCatalog = map[string]WAFRuleDetail{
	"path-traversal": {
		ID: "path-traversal", Name: "Path traversal", Phase: "path-traversal",
		Source: "demo", Type: "regex",
		Description: "演示数据：路径遍历类探测（如 /../etc/passwd）",
	},
	"sql-injection-uri": {
		ID: "sql-injection-uri", Name: "SQL injection (URI)", Phase: "sql-injection-uri",
		Source: "demo", Type: "regex", Targets: []string{"uri"},
		Description: "演示数据：URI 中的 SQL 注入特征",
	},
	"scanner-ua": {
		ID: "scanner-ua", Name: "Scanner User-Agent", Phase: "scanner-ua",
		Source: "demo", Type: "contains", Targets: []string{"headers"},
		Pattern: `(?i)(scanner|nikto|sqlmap|acunetix)`,
		Description: "演示数据：常见扫描器 User-Agent；试匹配需填写 User-Agent 且当前配置启用 WAF",
	},
	"ip-deny": {
		ID: "ip-deny", Name: "IP deny list", Phase: "ip deny",
		Source: "demo", Type: "ip",
		Description: "演示数据：客户端 IP 在 deny 列表中",
	},
	"suspicious-method": {
		ID: "suspicious-method", Name: "Suspicious HTTP method", Phase: "suspicious-method",
		Source: "demo",
		Description: "演示数据：非常规 HTTP 方法",
	},
}

// LookupWAFRule resolves event.rule to human-readable detail from config and builtins.
func LookupWAFRule(cfg *ingcore.Config, ruleField string) *WAFRuleDetail {
	ruleField = strings.TrimSpace(ruleField)
	if ruleField == "" {
		return nil
	}

	phase := ruleField
	id := ruleField
	if strings.HasPrefix(phase, "sig ") {
		id = strings.TrimSpace(strings.TrimPrefix(phase, "sig "))
		phase = "signature"
	}

	switch strings.ToLower(ruleField) {
	case "ip deny":
		return &WAFRuleDetail{
			ID: "ip-deny", Name: "IP deny list", Phase: "ip deny", Source: "phase", Type: "ip",
			Description: "客户端 IP 命中全局或规则级 waf.deny 列表",
		}
	case "ip allow":
		return &WAFRuleDetail{
			ID: "ip-allow", Name: "IP allow gate", Phase: "ip allow", Source: "phase", Type: "ip",
			Description: "客户端 IP 不在 waf.allow 白名单内",
		}
	}

	if cfg != nil {
		for _, wr := range cfg.WAF.Rules {
			if wr.ID == id || wr.ID == ruleField || wr.Name == ruleField {
				return ruleDetailFromConfig(wr)
			}
		}
		for _, wr := range waf.StarterRules() {
			if wr.ID == id {
				return ruleDetailFromStarter(wr, cfg)
			}
		}
	}

	if d, ok := demoRuleCatalog[id]; ok {
		copy := d
		return &copy
	}
	if d, ok := demoRuleCatalog[ruleField]; ok {
		copy := d
		return &copy
	}

	return &WAFRuleDetail{
		ID:          id,
		Name:        ruleField,
		Phase:       phase,
		Source:      "unknown",
		Description: "未在当前 ingress 配置或内置规则中找到定义；可能为历史日志或演示库事件",
	}
}

func ruleDetailFromConfig(wr rule.WAFRule) *WAFRuleDetail {
	targets := append([]string(nil), wr.Targets...)
	return &WAFRuleDetail{
		ID:          wr.ID,
		Name:        wr.Name,
		Phase:       "signature",
		Type:        wr.Type,
		Pattern:     wr.Pattern,
		Targets:     targets,
		Source:      "config",
		LogOnly:     wr.LogOnly,
		Enabled:     waf.RuleActive(wr),
		Description: "配置 waf.rules 中的自定义规则",
	}
}

func ruleDetailFromStarter(wr rule.WAFRule, cfg *ingcore.Config) *WAFRuleDetail {
	targets := append([]string(nil), wr.Targets...)
	enabled := true
	if cfg != nil {
		enabled = waf.BuiltinRuleEnabled(cfg.WAF.DisableBuiltin, cfg.WAF.BuiltinRules, wr.ID)
	}
	return &WAFRuleDetail{
		ID:          wr.ID,
		Name:        wr.Name,
		Phase:       "signature",
		Type:        wr.Type,
		Pattern:     wr.Pattern,
		Targets:     targets,
		Source:      "builtin",
		Builtin:     true,
		Enabled:     enabled,
		Description: "内置 starter 规则；可通过 waf.builtin_rules 或 disable_builtin 控制",
	}
}

// ListWAFRulesCatalog returns all known WAF rule definitions for tooltips and lookups.
func ListWAFRulesCatalog(cfg *ingcore.Config) []WAFRuleDetail {
	seen := make(map[string]struct{})
	var out []WAFRuleDetail
	add := func(d WAFRuleDetail) {
		if d.ID == "" {
			return
		}
		if _, ok := seen[d.ID]; ok {
			return
		}
		seen[d.ID] = struct{}{}
		out = append(out, d)
	}

	if cfg != nil {
		for _, wr := range cfg.WAF.Rules {
			add(*ruleDetailFromConfig(wr))
		}
		for _, wr := range waf.StarterRules() {
			add(*ruleDetailFromStarter(wr, cfg))
		}
	} else {
		for _, wr := range waf.StarterRules() {
			add(*ruleDetailFromStarter(wr, nil))
		}
	}

	for _, d := range demoRuleCatalog {
		add(d)
	}
	if d := LookupWAFRule(cfg, "ip deny"); d != nil {
		add(*d)
	}
	if d := LookupWAFRule(cfg, "ip allow"); d != nil {
		add(*d)
	}
	return out
}

// WAFRulesCatalog returns all known WAF rule definitions for tooltips and lookups.
func WAFRulesCatalog(cfg *ingcore.Config) []WAFRuleDetail {
	seen := make(map[string]struct{})
	var out []WAFRuleDetail
	add := func(d WAFRuleDetail) {
		if d.ID == "" {
			return
		}
		if _, ok := seen[d.ID]; ok {
			return
		}
		seen[d.ID] = struct{}{}
		out = append(out, d)
	}

	if cfg != nil {
		for _, wr := range cfg.WAF.Rules {
			add(*ruleDetailFromConfig(wr))
		}
		for _, wr := range waf.StarterRules() {
			add(*ruleDetailFromStarter(wr, cfg))
		}
	} else {
		for _, wr := range waf.StarterRules() {
			add(*ruleDetailFromStarter(wr, nil))
		}
	}

	if d := LookupWAFRule(cfg, "ip deny"); d != nil {
		add(*d)
	}
	if d := LookupWAFRule(cfg, "ip allow"); d != nil {
		add(*d)
	}
	for _, d := range demoRuleCatalog {
		add(d)
	}
	return out
}

func BuildWAFEventDetail(cfg *ingcore.Config, ev *model.WAFEvent) WAFEventDetail {
	if ev == nil {
		return WAFEventDetail{}
	}
	detail := LookupWAFRule(cfg, ev.Rule)
	note := "试匹配使用当前 ingress 配置与运行时 WAF 开关，与历史/演示事件可能不一致。"
	if detail != nil && detail.Source == "demo" {
		note = "该事件来自 admin 演示种子数据；当前配置文件未定义此规则，试匹配通常不会复现命中。"
	}
	return WAFEventDetail{
		ID:         ev.ID,
		Action:     ev.Action,
		Rule:       ev.Rule,
		Host:       ev.Host,
		Path:       ev.Path,
		ClientIP:   ev.ClientIP,
		UserAgent:  ev.UserAgent,
		CreatedAt:  ev.CreatedAt.Format(time.RFC3339),
		RuleDetail: detail,
		ReplayNote: note,
	}
}
