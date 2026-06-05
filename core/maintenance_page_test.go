package core

import (
	"strings"
	"testing"
	"time"
)

func TestMaintenancePageCopy_DefaultChinese(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 6, 5, 18, 30, 0, 0, time.FixedZone("CST", 8*3600))
	title, subtitle := maintenancePageCopy(compiledMaintenanceSettings{}, compiledMaintenanceWindow{
		configured: true,
		end:        end,
	})
	if title != defaultMaintenanceTitle {
		t.Fatalf("title: got %q want %q", title, defaultMaintenanceTitle)
	}
	wantSub := "我们正在进行计划维护，预计恢复时间：2026-06-05 18:30:00"
	if subtitle != wantSub {
		t.Fatalf("subtitle: got %q want %q", subtitle, wantSub)
	}
}

func TestMaintenancePageCopy_CustomTitleSubtitle(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 6, 5, 18, 30, 0, 0, time.UTC)
	title, subtitle := maintenancePageCopy(compiledMaintenanceSettings{
		Title:    "自定义标题",
		Subtitle: "自定义说明",
	}, compiledMaintenanceWindow{configured: true, end: end})
	if title != "自定义标题" || subtitle != "自定义说明" {
		t.Fatalf("got title=%q subtitle=%q", title, subtitle)
	}
}

func TestMaintenancePageCopy_NoWindowEnd(t *testing.T) {
	t.Parallel()
	_, subtitle := maintenancePageCopy(compiledMaintenanceSettings{}, compiledMaintenanceWindow{})
	if subtitle != "我们正在进行计划维护" {
		t.Fatalf("subtitle: got %q", subtitle)
	}
}

func TestIngressMaintenancePageHTML_DefaultCopy(t *testing.T) {
	t.Parallel()
	html := ingressMaintenancePageHTML(defaultMaintenanceTitle, "我们正在进行计划维护，预计恢复时间：2026-06-05 18:30:00", false, "", "", "", "")
	for _, want := range []string{
		"lang=\"zh-CN\"",
		defaultMaintenanceTitle,
		"我们正在进行计划维护，预计恢复时间：2026-06-05 18:30:00",
		">维护<",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in html", want)
		}
	}
	if strings.Contains(html, ">503<") {
		t.Fatal("maintenance page should not prominently show 503")
	}
}

func TestIngressMaintenancePageJSON(t *testing.T) {
	t.Parallel()
	raw := ingressMaintenancePageJSON(defaultMaintenanceTitle, "我们正在进行计划维护")
	if !strings.Contains(raw, `"status":"maintenance"`) {
		t.Fatalf("unexpected json: %s", raw)
	}
	if !strings.Contains(raw, defaultMaintenanceTitle) {
		t.Fatalf("unexpected json: %s", raw)
	}
}
