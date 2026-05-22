package service

import (
	"sort"
	"strings"

	ingcore "github.com/go-zoox/ingress/core"
)

// CacheOverview aggregates global + route HTTP cache config and access-log stats.
type CacheOverview struct {
	Global ingcore.CacheGlobalView `json:"global"`
	Routes []ingcore.CacheRouteRow `json:"routes"`
	Stats  CacheStats              `json:"stats"`
}

type CacheStats struct {
	TotalRequests int            `json:"total_requests"`
	CacheHits     int            `json:"cache_hits"`
	HitRate       float64        `json:"hit_rate"`
	TopHosts      []NamedHitRate `json:"top_hosts"`
	TopPaths      []PathHitRate  `json:"top_paths"`
}

type NamedHitRate struct {
	Host    string  `json:"host"`
	Hits    int     `json:"hits"`
	Total   int     `json:"total"`
	HitRate float64 `json:"hit_rate"`
}

type PathHitRate struct {
	Path    string  `json:"path"`
	Hits    int     `json:"hits"`
	Total   int     `json:"total"`
	HitRate float64 `json:"hit_rate"`
}

// Cache builds cache dashboard data.
type Cache struct {
	ingress *Ingress
	logs    *Logs
}

func NewCache(ingress *Ingress, logs *Logs) *Cache {
	return &Cache{ingress: ingress, logs: logs}
}

func (c *Cache) Overview() (CacheOverview, error) {
	out := CacheOverview{
		Routes: []ingcore.CacheRouteRow{},
		Stats: CacheStats{
			TopHosts: []NamedHitRate{},
			TopPaths: []PathHitRate{},
		},
	}
	icfg, err := c.ingress.LoadConfig()
	if err != nil {
		return out, err
	}
	out.Global = ingcore.CacheGlobalViewFromConfig(icfg)
	rows, err := ingcore.ListCacheRouteRows(icfg)
	if err != nil {
		return out, err
	}
	if rows == nil {
		rows = []ingcore.CacheRouteRow{}
	}
	out.Routes = rows
	out.Stats = c.statsFromAccessLog()
	return out, nil
}

func (c *Cache) statsFromAccessLog() CacheStats {
	empty := CacheStats{
		TopHosts: []NamedHitRate{},
		TopPaths: []PathHitRate{},
	}
	lines, err := c.logs.TailAccess(5000)
	if err != nil || len(lines) == 0 {
		return empty
	}
	hostTotal := map[string]int{}
	hostHits := map[string]int{}
	pathTotal := map[string]int{}
	pathHits := map[string]int{}
	total := 0
	hits := 0
	for _, line := range lines {
		e, ok := parseAccessLine(line)
		if !ok {
			continue
		}
		total++
		if e.CacheHit {
			hits++
		}
		hostTotal[e.Host]++
		if e.CacheHit {
			hostHits[e.Host]++
		}
		pathKey := e.Path
		if pathKey == "" {
			pathKey = "/"
		}
		if i := strings.Index(pathKey, "?"); i >= 0 {
			pathKey = pathKey[:i]
		}
		pathTotal[pathKey]++
		if e.CacheHit {
			pathHits[pathKey]++
		}
	}
	rate := 0.0
	if total > 0 {
		rate = float64(hits) / float64(total) * 100
	}
	return CacheStats{
		TotalRequests: total,
		CacheHits:     hits,
		HitRate:       rate,
		TopHosts:      topHostHitRates(hostTotal, hostHits, 8),
		TopPaths:      topPathHitRates(pathTotal, pathHits, 8),
	}
}

func topHostHitRates(total, hits map[string]int, n int) []NamedHitRate {
	keys := topHitRateKeys(total, n)
	out := make([]NamedHitRate, 0, len(keys))
	for _, k := range keys {
		t := total[k]
		hh := hits[k]
		rate := 0.0
		if t > 0 {
			rate = float64(hh) / float64(t) * 100
		}
		out = append(out, NamedHitRate{Host: k, Hits: hh, Total: t, HitRate: rate})
	}
	return out
}

func topPathHitRates(total, hits map[string]int, n int) []PathHitRate {
	keys := topHitRateKeys(total, n)
	out := make([]PathHitRate, 0, len(keys))
	for _, k := range keys {
		t := total[k]
		hh := hits[k]
		rate := 0.0
		if t > 0 {
			rate = float64(hh) / float64(t) * 100
		}
		out = append(out, PathHitRate{Path: k, Hits: hh, Total: t, HitRate: rate})
	}
	return out
}

func topHitRateKeys(total map[string]int, n int) []string {
	type pair struct {
		key   string
		total int
	}
	all := make([]pair, 0, len(total))
	for k, t := range total {
		all = append(all, pair{k, t})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].total == all[j].total {
			return all[i].key < all[j].key
		}
		return all[i].total > all[j].total
	})
	if n > len(all) {
		n = len(all)
	}
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = all[i].key
	}
	return keys
}
