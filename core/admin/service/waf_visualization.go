package service

import (
	"sort"

	"github.com/go-zoox/ingress/core/admin/model"
	"github.com/go-zoox/ingress/core/admin/service/geoip"
)

// WAFAttackPoint aggregates WAF events at a geographic location.
type WAFAttackPoint struct {
	Lat    float64  `json:"lat"`
	Lng    float64  `json:"lng"`
	Label  string   `json:"label"`
	Count  int      `json:"count"`
	Block  int      `json:"block"`
	Audit  int      `json:"audit"`
	IPs    []string `json:"ips"`
	Approx bool     `json:"approx,omitempty"`
}

// WAFVisualization is the payload for the WAF attack map.
type WAFVisualization struct {
	Points     []WAFAttackPoint `json:"points"`
	Total      int              `json:"total"`
	UnknownIPs int              `json:"unknown_ips"`
	Server     GeoPoint         `json:"server"`
	GeoIP      geoip.Status     `json:"geoip"`
}

// GeoPoint is a lat/lng label for JSON responses.
type GeoPoint struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Label string  `json:"label"`
}

// BuildWAFVisualization aggregates WAF events by client IP geo location.
func BuildWAFVisualization(events []model.WAFEvent) WAFVisualization {
	type bucket struct {
		geo    geoip.Point
		block  int
		audit  int
		ips    map[string]struct{}
	}

	buckets := map[string]*bucket{}
	unknown := 0

	for _, ev := range events {
		geo, ok := geoip.Lookup(ev.ClientIP)
		if !ok {
			unknown++
			continue
		}
		key := geo.Label
		if geo.Approx {
			key = ev.ClientIP
		}
		b, exists := buckets[key]
		if !exists {
			b = &bucket{geo: geo, ips: map[string]struct{}{}}
			buckets[key] = b
		}
		b.ips[ev.ClientIP] = struct{}{}
		if ev.Action == "audit" {
			b.audit++
		} else {
			b.block++
		}
	}

	points := make([]WAFAttackPoint, 0, len(buckets))
	for _, b := range buckets {
		ips := make([]string, 0, len(b.ips))
		for ip := range b.ips {
			ips = append(ips, ip)
		}
		sort.Strings(ips)
		points = append(points, WAFAttackPoint{
			Lat:    b.geo.Lat,
			Lng:    b.geo.Lng,
			Label:  b.geo.Label,
			Count:  b.block + b.audit,
			Block:  b.block,
			Audit:  b.audit,
			IPs:    ips,
			Approx: b.geo.Approx,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Count > points[j].Count
	})

	ingress := geoip.GlobalIngress()
	return WAFVisualization{
		Points:     points,
		Total:      len(events),
		UnknownIPs: unknown,
		Server: GeoPoint{
			Lat:   ingress.Lat,
			Lng:   ingress.Lng,
			Label: ingress.Label,
		},
		GeoIP: geoip.GlobalStatus(),
	}
}
