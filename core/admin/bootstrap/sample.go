package bootstrap

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

// seedSampleDataIfEmpty inserts example rows when the DB has no WAF events yet.
// Used by examples/admin-console on first start (admin.db); not a runtime demo fallback.
func seedSampleDataIfEmpty() error {
	db := gormx.GetDB()
	var n int64
	if err := db.Model(&model.WAFEvent{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	end := time.Now()
	start := end.AddDate(0, -3, 0)
	rng := rand.New(rand.NewSource(42))

	wafRules := []weightedWAF{
		{"block", "path-traversal", "waf-demo.example.com", "/admin", 12},
		{"block", "path-traversal", "waf-demo.example.com", "/../etc/passwd", 8},
		{"block", "sql-injection-uri", "api.example.com", "/search?q=1' OR '1'='1", 10},
		{"block", "sql-injection-uri", "waf-demo.example.com", "/search?q=union+select", 6},
		{"audit", "scanner-ua", "api.example.com", "/", 15},
		{"audit", "scanner-ua", "cdn.example.com", "/assets/app.js", 8},
		{"block", "ip-deny", "api.example.com", "/api/users", 5},
		{"audit", "suspicious-method", "admin.internal", "/healthz", 4},
	}

	waf := make([]model.WAFEvent, 0, 180)
	for i := 0; i < 180; i++ {
		pick := weightedPick(rng, wafRules)
		waf = append(waf, model.WAFEvent{
			Action:    pick.action,
			Rule:      pick.rule,
			Host:      pick.host,
			Path:      pick.path,
			ClientIP:  sampleIP(rng),
			CreatedAt: randomTime(rng, start, end, i, 180),
		})
	}
	if err := db.Create(&waf).Error; err != nil {
		return fmt.Errorf("seed waf events: %w", err)
	}

	auditActions := []struct {
		action string
		detail string
		gap    time.Duration
	}{
		{"ingress.reload", "examples/admin-console/ingress.yaml", 12 * time.Hour},
		{"config.validate", "ingress.yaml ok", 6 * time.Hour},
		{"config.save", "hash=a1b2c3d4 rules=6", 3 * 24 * time.Hour},
		{"ingress.reload", "SIGHUP pid=12847", 8 * time.Hour},
		{"config.validate", "ingress.yaml ok", 4 * time.Hour},
		{"config.save", "hash=e5f6a7b8 waf.enabled=true", 5 * 24 * time.Hour},
		{"ingress.reload", "examples/admin-console/ingress.yaml", 18 * time.Hour},
		{"config.validate", "rules[3].host: ok", 2 * time.Hour},
	}

	audit := make([]model.AuditLog, 0, len(auditActions))
	at := start.Add(2 * time.Hour)
	for _, row := range auditActions {
		audit = append(audit, model.AuditLog{
			Action:    row.action,
			Detail:    row.detail,
			Actor:     "admin",
			CreatedAt: at,
		})
		at = at.Add(row.gap + time.Duration(rng.Intn(3600))*time.Second)
		if at.After(end) {
			at = end.Add(-time.Duration(rng.Intn(7200)) * time.Second)
		}
	}
	if err := db.Create(&audit).Error; err != nil {
		return fmt.Errorf("seed audit log: %w", err)
	}

	revisions := []model.ConfigRevision{
		{Hash: "a1b2c3d4", Note: "initial sample", CreatedAt: start.Add(24 * time.Hour)},
		{Hash: "c3d4e5f6", Note: "enable waf builtin", CreatedAt: start.Add(18 * 24 * time.Hour)},
		{Hash: "e5f6a7b8", Note: "add cdn wildcard", CreatedAt: start.Add(35 * 24 * time.Hour)},
		{Hash: "f7a8b9c0", Note: "tunnel regex tuning", CreatedAt: start.Add(52 * 24 * time.Hour)},
		{Hash: "1a2b3c4d", Note: "logging transports", CreatedAt: start.Add(68 * 24 * time.Hour)},
		{Hash: "9f8e7d6c", Note: "healthcheck interval", CreatedAt: end.Add(-10 * 24 * time.Hour)},
	}
	for i := range revisions {
		revisions[i].Content = "# sample ingress revision\nversion: v1\n"
	}
	if err := db.Create(&revisions).Error; err != nil {
		return fmt.Errorf("seed config revisions: %w", err)
	}

	return nil
}

type weightedWAF struct {
	action string
	rule   string
	host   string
	path   string
	weight int
}

func weightedPick(rng *rand.Rand, rules []weightedWAF) weightedWAF {
	total := 0
	for _, r := range rules {
		total += r.weight
	}
	n := rng.Intn(total)
	for _, r := range rules {
		n -= r.weight
		if n < 0 {
			return r
		}
	}
	return rules[0]
}

func sampleIP(rng *rand.Rand) string {
	ips := []string{
		"203.0.113.44", "198.51.100.8", "192.0.2.99", "203.0.113.12",
		"198.51.100.22", "203.0.113.88", "192.0.2.17", "10.0.0.5",
	}
	return ips[rng.Intn(len(ips))]
}

func randomTime(rng *rand.Rand, start, end time.Time, idx, total int) time.Time {
	// Spread evenly with jitter so charts show long-running history.
	span := end.Sub(start)
	base := start.Add(time.Duration(idx) * span / time.Duration(total))
	jitter := time.Duration(rng.Intn(int(span/time.Duration(total*2)))) - span/time.Duration(total*4)
	t := base.Add(jitter)
	if t.Before(start) {
		return start.Add(time.Duration(rng.Intn(3600)) * time.Second)
	}
	if t.After(end) {
		return end.Add(-time.Duration(rng.Intn(3600)) * time.Second)
	}
	return t
}
