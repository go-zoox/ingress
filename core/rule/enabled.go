package rule

// IsEnabled reports whether the rule participates in routing (default true when enabled is omitted).
func (r *Rule) IsEnabled() bool {
	if r == nil || r.Enabled == nil {
		return true
	}
	return *r.Enabled
}
