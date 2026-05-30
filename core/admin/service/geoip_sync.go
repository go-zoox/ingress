package service

import (
	"github.com/go-zoox/logger"
	"github.com/go-zoox/ingress/core/admin/service/geoip"
)

// SyncGeoIPFromIngress reloads MaxMind GeoIP from the current ingress.yaml on disk.
func SyncGeoIPFromIngress(ing *Ingress) {
	if ing == nil {
		return
	}
	icfg, err := ing.LoadConfig()
	if err != nil {
		logger.Warnf("admin geoip: sync skipped: %v", err)
		return
	}
	g := icfg.Admin.GeoIP
	if _, err := geoip.Reconfigure(geoip.Config{
		Database:     g.Database,
		IngressLat:   g.IngressLat,
		IngressLng:   g.IngressLng,
		IngressLabel: g.IngressLabel,
	}); err != nil {
		logger.Warnf("admin geoip: sync failed: %v", err)
	}
}
