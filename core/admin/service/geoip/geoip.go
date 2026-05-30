package geoip

import (
	"net"
	"strings"
	"sync"

	"github.com/go-zoox/logger"
	"github.com/oschwald/geoip2-golang"
)

// Point is a resolved latitude/longitude for map visualization.
type Point struct {
	Lat    float64
	Lng    float64
	Label  string
	Approx bool
}

// IngressPoint is the defender / ingress location on the map.
type IngressPoint struct {
	Lat   float64
	Lng   float64
	Label string
}

// Config configures MaxMind GeoLite2 lookup for WAF attack maps.
type Config struct {
	Database     string
	IngressLat   float64
	IngressLng   float64
	IngressLabel string
}

// Status describes runtime GeoIP availability for the admin API.
type Status struct {
	// Configured is true when admin.geoip.database is set (non-empty).
	Configured bool `json:"configured"`
	// Enabled is true only when MaxMind is loaded and active (same as Loaded).
	Enabled bool `json:"enabled"`
	Loaded  bool   `json:"loaded"`
	Source  string `json:"source"` // maxmind | fallback
	Database string `json:"database,omitempty"`
	Error    string `json:"error,omitempty"`
	Reason   string `json:"reason,omitempty"` // unset | not_found | permission_denied | invalid | open_failed
}

var (
	globalMu sync.RWMutex
	global   *Service
)

// Service resolves client IPs using MaxMind when configured, else fallback rules.
type Service struct {
	reader  *geoip2.Reader
	cfg     Config
	status  Status
	ingress IngressPoint
}

// Reconfigure closes any existing reader and re-initializes GeoIP from cfg.
func Reconfigure(cfg Config) (*Service, error) {
	globalMu.Lock()
	old := global
	globalMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return Init(cfg)
}

// Init opens the GeoLite2 database when configured. Missing or invalid DB falls back without failing startup.
func Init(cfg Config) (*Service, error) {
	s := &Service{
		cfg:     cfg,
		ingress: defaultIngress(cfg),
		status:  Status{Source: "fallback", Reason: ReasonUnset},
	}

	path := strings.TrimSpace(cfg.Database)
	if path == "" {
		setGlobal(s)
		return s, nil
	}

	s.status.Configured = true
	s.status.Database = path

	if ok, reason, msg := DatabaseCheck(path); !ok {
		s.status.Reason = reason
		s.status.Error = msg
		s.status.Source = "fallback"
		s.status.Loaded = false
		s.status.Enabled = false
		if msg != "" {
			logger.Warnf("admin geoip: %s; MaxMind disabled, WAF map uses approximate lookup", msg)
		}
		setGlobal(s)
		return s, nil
	}

	reader, err := geoip2.Open(path)
	if err != nil {
		s.status.Reason = openFailureReason(err)
		s.status.Error = openFailureMessage(path, err)
		s.status.Source = "fallback"
		s.status.Loaded = false
		s.status.Enabled = false
		logger.Warnf("admin geoip: %s; MaxMind disabled, WAF map uses approximate lookup", s.status.Error)
		setGlobal(s)
		return s, nil
	}

	s.reader = reader
	s.status.Loaded = true
	s.status.Enabled = true
	s.status.Source = "maxmind"
	s.status.Reason = ""
	s.status.Error = ""
	logger.Infof("admin geoip: loaded MaxMind database %s", path)
	setGlobal(s)
	return s, nil
}

// Close releases the MaxMind reader.
func (s *Service) Close() error {
	if s == nil || s.reader == nil {
		return nil
	}
	return s.reader.Close()
}

// Status returns the current GeoIP status snapshot.
func (s *Service) Status() Status {
	if s == nil {
		return Status{Source: "fallback"}
	}
	return s.status
}

// Ingress returns the configured ingress map anchor.
func (s *Service) Ingress() IngressPoint {
	if s == nil {
		return defaultIngress(Config{})
	}
	return s.ingress
}

// Lookup resolves ip to a map point. Private / loopback IPs return ok=false.
func Lookup(ip string) (Point, bool) {
	globalMu.RLock()
	s := global
	globalMu.RUnlock()
	if s != nil {
		if p, ok := s.lookup(ip); ok {
			return p, true
		}
	}
	return lookupFallback(ip)
}

// GlobalStatus returns GeoIP status from the global service.
func GlobalStatus() Status {
	globalMu.RLock()
	s := global
	globalMu.RUnlock()
	if s == nil {
		return Status{Source: "fallback"}
	}
	return s.Status()
}

// GlobalIngress returns ingress coordinates from the global service.
func GlobalIngress() IngressPoint {
	globalMu.RLock()
	s := global
	globalMu.RUnlock()
	if s == nil {
		return defaultIngress(Config{})
	}
	return s.Ingress()
}

func setGlobal(s *Service) {
	globalMu.Lock()
	global = s
	globalMu.Unlock()
}

func (s *Service) lookup(ip string) (Point, bool) {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "-" {
		return Point{}, false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Point{}, false
	}
	if parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast() || parsed.IsUnspecified() {
		return Point{}, false
	}

	if s.reader != nil {
		if p, ok := s.lookupMaxMind(parsed, ip); ok {
			return p, true
		}
	}
	return lookupFallback(ip)
}

func (s *Service) lookupMaxMind(ip net.IP, raw string) (Point, bool) {
	record, err := s.reader.City(ip)
	if err != nil || record == nil {
		return Point{}, false
	}
	lat := record.Location.Latitude
	lng := record.Location.Longitude
	if lat == 0 && lng == 0 {
		return Point{}, false
	}

	label := pickName(record.City.Names)
	if label == "" {
		label = pickName(record.Country.Names)
	}
	if label == "" && len(record.Subdivisions) > 0 {
		label = pickName(record.Subdivisions[0].Names)
	}
	if label == "" {
		label = raw
	}

	return Point{Lat: lat, Lng: lng, Label: label, Approx: false}, true
}

func pickName(names map[string]string) string {
	if names == nil {
		return ""
	}
	if v := strings.TrimSpace(names["zh-CN"]); v != "" {
		return v
	}
	if v := strings.TrimSpace(names["en"]); v != "" {
		return v
	}
	for _, v := range names {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func defaultIngress(cfg Config) IngressPoint {
	lat, lng := cfg.IngressLat, cfg.IngressLng
	label := strings.TrimSpace(cfg.IngressLabel)
	if lat == 0 && lng == 0 {
		lat, lng = 31.2304, 121.4737
	}
	if label == "" {
		label = "Ingress"
	}
	return IngressPoint{Lat: lat, Lng: lng, Label: label}
}
