package bootstrap

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-zoox/gormx"
	"github.com/go-zoox/ingress/core/admin/model"
)

const seedWAFEventCount = 360

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
		{"block", "path-traversal", "waf-demo.example.com", "/admin", 14},
		{"block", "path-traversal", "waf-demo.example.com", "/../etc/passwd", 10},
		{"block", "sql-injection-uri", "api.example.com", "/search?q=1' OR '1'='1", 12},
		{"block", "sql-injection-uri", "waf-demo.example.com", "/search?q=union+select", 8},
		{"block", "sql-injection-uri", "api.example.com", "/api/report?id=1;DROP+TABLE", 6},
		{"audit", "scanner-ua", "api.example.com", "/", 18},
		{"audit", "scanner-ua", "cdn.example.com", "/assets/app.js", 10},
		{"audit", "scanner-ua", "waf-demo.example.com", "/.env", 7},
		{"block", "ip-deny", "api.example.com", "/api/users", 6},
		{"block", "ip-deny", "api.example.com", "/api/admin", 4},
		{"audit", "suspicious-method", "admin.internal", "/healthz", 5},
		{"audit", "suspicious-method", "api.example.com", "/debug", 3},
	}

	scannerUAs := []string{
		"Mozilla/5.0 (compatible; Nikto/2.1.6)",
		"sqlmap/1.7.2#stable",
		"Acunetix-Web-Security-Scanner",
		"Mozilla/5.0 (compatible; Nmap Scripting Engine)",
		"python-requests/2.31.0 scanner-probe",
	}

	waf := make([]model.WAFEvent, 0, seedWAFEventCount)
	for i := 0; i < seedWAFEventCount; i++ {
		pick := weightedPick(rng, wafRules)
		ev := model.WAFEvent{
			Action:    pick.action,
			Rule:      pick.rule,
			Host:      pick.host,
			Path:      pick.path,
			ClientIP:  sampleIP(rng),
			CreatedAt: randomTime(rng, start, end, i, seedWAFEventCount),
		}
		if pick.rule == "scanner-ua" {
			ev.UserAgent = scannerUAs[rng.Intn(len(scannerUAs))]
		}
		waf = append(waf, ev)
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

// sampleGeoIPs uses RFC5737 TEST-NET addresses; geo labels live in service/geoip/fallback.go.
var sampleGeoIPs = []weightedIP{
	{"203.0.113.44", 16},  // 北京
	{"203.0.113.101", 14}, // 香港
	{"203.0.113.102", 10}, // 台北
	{"203.0.113.77", 12},  // 首尔
	{"198.51.100.22", 14}, // 东京
	{"203.0.113.55", 12},  // 新加坡
	{"203.0.113.66", 10},  // 孟买
	{"192.0.2.22", 8},     // 曼谷
	{"192.0.2.33", 8},     // 雅加达
	{"192.0.2.44", 6},     // 马尼拉
	{"203.0.113.88", 10},  // 伦敦
	{"203.0.113.33", 9},   // 法兰克福
	{"203.0.113.99", 8},   // 阿姆斯特丹
	{"198.51.100.11", 9},  // 巴黎
	{"198.51.100.77", 6},  // 斯德哥尔摩
	{"192.0.2.77", 5},     // 爱丁堡
	{"203.0.113.21", 11},  // 纽约
	{"192.0.2.99", 10},    // 旧金山
	{"192.0.2.55", 7},     // 温哥华
	{"198.51.100.33", 7},  // 多伦多
	{"203.0.113.12", 9},   // 圣保罗
	{"198.51.100.88", 6},  // 布宜诺斯艾利斯
	{"192.0.2.11", 7},     // 墨西哥城
	{"198.51.100.8", 8},   // 莫斯科
	{"192.0.2.66", 6},     // 伊斯坦布尔
	{"198.51.100.44", 7},  // 迪拜
	{"198.51.100.55", 5},  // 开罗
	{"198.51.100.66", 5},  // 约翰内斯堡
	{"192.0.2.17", 9},     // 悉尼
	{"10.0.0.5", 4},       // 内网（不显示在地球上）
}

type weightedIP struct {
	ip     string
	weight int
}

func sampleIP(rng *rand.Rand) string {
	total := 0
	for _, row := range sampleGeoIPs {
		total += row.weight
	}
	n := rng.Intn(total)
	for _, row := range sampleGeoIPs {
		n -= row.weight
		if n < 0 {
			return row.ip
		}
	}
	return sampleGeoIPs[0].ip
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
