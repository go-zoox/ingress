package service

import (
	"fmt"
	"strings"

	ingcore "github.com/go-zoox/ingress/core"
)

var globalModuleLabels = map[string]string{
	"general":      "基础",
	"admin":        "Admin",
	"cache":        "全局缓存",
	"logging":      "日志",
	"waf":          "WAF",
	"healthcheck":  "健康检查",
	"https":        "TLS / HTTPS",
	"fallback":     "Fallback",
	"other":        "其他",
}

func globalTouchesFromModules(modules []string) []string {
	out := make([]string, 0, len(modules))
	for _, id := range modules {
		if id == "rules" {
			continue
		}
		if label, ok := globalModuleLabels[id]; ok {
			out = append(out, label)
		} else {
			out = append(out, id)
		}
	}
	return out
}

type routeRowKey struct {
	ruleIndex int
	pathIndex int
}

func routeKey(row ingcore.RouteRow) routeRowKey {
	return routeRowKey{ruleIndex: row.RuleIndex, pathIndex: row.PathIndex}
}

func rowFingerprint(row ingcore.RouteRow) string {
	return strings.Join([]string{
		row.Host,
		row.HostType,
		row.Path,
		row.BackendType,
		row.Target,
		row.WAF,
		fmt.Sprintf("cache:%v", row.Cache),
		row.Auth,
		row.HealthCheck,
	}, "|")
}

func compareRouteRows(before, after ingcore.RouteRow) []string {
	var fields []string
	if before.Host != after.Host {
		fields = append(fields, "host")
	}
	if before.Path != after.Path {
		fields = append(fields, "path")
	}
	if before.BackendType != after.BackendType {
		fields = append(fields, "backend_type")
	}
	if before.Target != after.Target {
		fields = append(fields, "target")
	}
	if before.WAF != after.WAF {
		fields = append(fields, "waf")
	}
	if before.Cache != after.Cache {
		fields = append(fields, "cache")
	}
	if before.Auth != after.Auth {
		fields = append(fields, "auth")
	}
	if before.HealthCheck != after.HealthCheck {
		fields = append(fields, "health_check")
	}
	return fields
}

// AnalyzeRouteImpacts diffs published vs draft ingress YAML for flattened route rows.
func AnalyzeRouteImpacts(ing *Ingress, published, draft string) ([]ConfigRouteImpact, error) {
	baseCfg, err := ing.LoadConfigFromYAML(published)
	if err != nil {
		return nil, fmt.Errorf("published config: %w", err)
	}
	draftCfg, err := ing.LoadConfigFromYAML(draft)
	if err != nil {
		return nil, fmt.Errorf("draft config: %w", err)
	}
	baseRows, err := ingcore.ListRouteRows(baseCfg)
	if err != nil {
		return nil, err
	}
	draftRows, err := ingcore.ListRouteRows(draftCfg)
	if err != nil {
		return nil, err
	}

	baseMap := make(map[routeRowKey]ingcore.RouteRow, len(baseRows))
	for _, r := range baseRows {
		baseMap[routeKey(r)] = r
	}
	draftMap := make(map[routeRowKey]ingcore.RouteRow, len(draftRows))
	for _, r := range draftRows {
		draftMap[routeKey(r)] = r
	}

	var out []ConfigRouteImpact
	for k, after := range draftMap {
		before, ok := baseMap[k]
		if !ok {
			out = append(out, ConfigRouteImpact{
				Kind:      "added",
				Host:      after.Host,
				Path:      after.Path,
				RuleIndex: after.RuleIndex,
				PathIndex: after.PathIndex,
				After:     after.Target,
			})
			continue
		}
		if rowFingerprint(before) != rowFingerprint(after) {
			fields := compareRouteRows(before, after)
			out = append(out, ConfigRouteImpact{
				Kind:      "changed",
				Host:      after.Host,
				Path:      after.Path,
				RuleIndex: after.RuleIndex,
				PathIndex: after.PathIndex,
				Fields:    fields,
				Before:    before.Target,
				After:     after.Target,
			})
		}
	}
	for k, before := range baseMap {
		if _, ok := draftMap[k]; ok {
			continue
		}
		out = append(out, ConfigRouteImpact{
			Kind:      "removed",
			Host:      before.Host,
			Path:      before.Path,
			RuleIndex: before.RuleIndex,
			PathIndex: before.PathIndex,
			Before:    before.Target,
		})
	}
	return out, nil
}
