package rule

import "testing"

func TestRuleIsEnabled(t *testing.T) {
	if !(&Rule{}).IsEnabled() {
		t.Fatal("default should be enabled")
	}
	on := true
	off := false
	if !(&Rule{Enabled: &on}).IsEnabled() {
		t.Fatal("explicit true")
	}
	if (&Rule{Enabled: &off}).IsEnabled() {
		t.Fatal("explicit false")
	}
}
