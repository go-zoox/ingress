package geoip

import (
	"net"
	"strings"
)

// demoIPGeo maps RFC5737 documentation addresses used in bootstrap seed data.
var demoIPGeo = map[string]Point{
	"203.0.113.44":  {Lat: 39.9042, Lng: 116.4074, Label: "北京"},
	"203.0.113.12":  {Lat: -23.5505, Lng: -46.6333, Label: "圣保罗"},
	"203.0.113.88":  {Lat: 51.5074, Lng: -0.1278, Label: "伦敦"},
	"203.0.113.21":  {Lat: 40.7128, Lng: -74.0060, Label: "纽约"},
	"203.0.113.33":  {Lat: 50.1109, Lng: 8.6821, Label: "法兰克福"},
	"203.0.113.55":  {Lat: 1.3521, Lng: 103.8198, Label: "新加坡"},
	"203.0.113.66":  {Lat: 19.0760, Lng: 72.8777, Label: "孟买"},
	"203.0.113.77":  {Lat: 37.5665, Lng: 126.9780, Label: "首尔"},
	"203.0.113.99":  {Lat: 52.3676, Lng: 4.9041, Label: "阿姆斯特丹"},
	"203.0.113.101": {Lat: 22.3193, Lng: 114.1694, Label: "香港"},
	"203.0.113.102": {Lat: 25.0330, Lng: 121.5654, Label: "台北"},
	"198.51.100.8":  {Lat: 55.7558, Lng: 37.6173, Label: "莫斯科"},
	"198.51.100.22": {Lat: 35.6762, Lng: 139.6503, Label: "东京"},
	"198.51.100.11": {Lat: 48.8566, Lng: 2.3522, Label: "巴黎"},
	"198.51.100.33": {Lat: 43.6532, Lng: -79.3832, Label: "多伦多"},
	"198.51.100.44": {Lat: 25.2048, Lng: 55.2708, Label: "迪拜"},
	"198.51.100.55": {Lat: 30.0444, Lng: 31.2357, Label: "开罗"},
	"198.51.100.66": {Lat: -26.2041, Lng: 28.0473, Label: "约翰内斯堡"},
	"198.51.100.77": {Lat: 59.3293, Lng: 18.0686, Label: "斯德哥尔摩"},
	"198.51.100.88": {Lat: -34.6037, Lng: -58.3816, Label: "布宜诺斯艾利斯"},
	"192.0.2.99":    {Lat: 37.7749, Lng: -122.4194, Label: "旧金山"},
	"192.0.2.17":    {Lat: -33.8688, Lng: 151.2093, Label: "悉尼"},
	"192.0.2.11":    {Lat: 19.4326, Lng: -99.1332, Label: "墨西哥城"},
	"192.0.2.22":    {Lat: 13.7563, Lng: 100.5018, Label: "曼谷"},
	"192.0.2.33":    {Lat: -6.2088, Lng: 106.8456, Label: "雅加达"},
	"192.0.2.44":    {Lat: 14.5995, Lng: 120.9842, Label: "马尼拉"},
	"192.0.2.55":    {Lat: 49.2827, Lng: -123.1207, Label: "温哥华"},
	"192.0.2.66":    {Lat: 41.0082, Lng: 28.9784, Label: "伊斯坦布尔"},
	"192.0.2.77":    {Lat: 55.9533, Lng: -3.1883, Label: "爱丁堡"},
}

func lookupFallback(ip string) (Point, bool) {
	ip = strings.TrimSpace(ip)
	if ip == "" || ip == "-" {
		return Point{}, false
	}
	if p, ok := demoIPGeo[ip]; ok {
		return p, true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return Point{}, false
	}
	if parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast() || parsed.IsUnspecified() {
		return Point{}, false
	}
	lat := (float64(int(parsed[15])|int(parsed[14])<<8)/65535)*160 - 80
	lng := (float64(int(parsed[13])|int(parsed[12])<<8)/65535)*360 - 180
	return Point{Lat: lat, Lng: lng, Label: ip, Approx: true}, true
}
