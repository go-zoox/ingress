package rule

// WAF configures Web Application Firewall defaults (top-level waf: in config).
// Per-route overlays live in rules[].waf as generic maps merged in core/waf_patch.go.
//
// disable_builtin — when true, embedded starter signatures are omitted (default false keeps them when Enabled).
type WAF struct {
	Enabled bool `config:"enabled"`

	TrustProxy bool `config:"trust_proxy"`

	// XFFIndex selects X-Forwarded-For segment when TrustProxy is true (0 = leftmost).
	XFFIndex int64 `config:"xff_index"`

	LogOnly bool `config:"log_only"`

	BlockStatusCode  int64  `config:"block_status_code"`
	BlockContentType string `config:"block_content_type"`
	BlockBody        string `config:"block_body"`

	DisableBuiltin bool `config:"disable_builtin"`

	// BuiltinRules optionally enables/disables individual starter rules by id.
	// When a key is absent, the rule follows disable_builtin (off when true, on when false).
	BuiltinRules map[string]bool `config:"builtin_rules"`

	// BuiltinRuleActions sets block / audit / pass per built-in rule id (overrides default block).
	BuiltinRuleActions map[string]string `config:"builtin_rule_actions"`

	Deny  []string  `config:"deny"`
	Allow []string  `config:"allow"`
	Rules []WAFRule `config:"rules"`
}

// WAFRule matches attack patterns against selected request parts (no body scanning in v1).
type WAFRule struct {
	ID      string   `config:"id"`
	Name    string   `config:"name"`
	// Action is block (default), audit (log only), or pass (allow on match, stop further signatures).
	// log_only: true is equivalent to action: audit when action is omitted.
	Action  string   `config:"action"`
	LogOnly bool     `config:"log_only"`
	Enabled *bool    `config:"enabled"`
	Type    string   `config:"type"`
	Pattern string   `config:"pattern"`
	Targets []string `config:"targets"`
}
