package waf

import (
	"net"
	"net/http"
	"strings"

	"github.com/go-zoox/logger"
)

// CheckRequest returns whether the proxy middleware should terminate with the WAF block response.
// If reportFn is non-nil, it is called for each block/audit hit with the action, rule/phase, and client IP.
func CheckRequest(p *Profile, r *http.Request, hostname, path, method string, reportFn func(action string, rule string, cliIP string)) bool {
	if p == nil || !p.Enabled || r == nil {
		return false
	}

	if hostMatchesAllowList(hostname, p.allowHosts) {
		return false
	}

	cli := clientIP(r, p.TrustProxy, p.XFFIndex)

	if len(p.denyNet) > 0 && ipMatchesNets(cli, p.denyNet) {
		stop := !p.GlobalLogOnly
		logHit(stop, "ip deny", hostname, path, method, cli, reportFn)
		return stop
	}

	if len(p.allowNet) > 0 && !ipMatchesNets(cli, p.allowNet) {
		stop := !p.GlobalLogOnly
		logHit(stop, "ip allow", hostname, path, method, cli, reportFn)
		return stop
	}

	rawQuery := r.URL.RawQuery
	for _, sr := range p.signatureRules {
		if hostMatchesAllowList(hostname, sr.skipHosts) {
			continue
		}
		if !matchesSignature(sr, r, path, rawQuery) {
			continue
		}
		act := resolveSigAction(sr, p.GlobalLogOnly)
		switch act {
		case ActionPass:
			logHitWithAction(ActionPass, "sig "+sr.id, hostname, path, method, cli, reportFn)
			return false
		case ActionAudit:
			logHitWithAction(ActionAudit, "sig "+sr.id, hostname, path, method, cli, reportFn)
			continue
		default:
			logHitWithAction(ActionBlock, "sig "+sr.id, hostname, path, method, cli, reportFn)
			return true
		}
	}
	return false
}

func logHit(stop bool, phase, hostname, path, method string, cli net.IP, reportFn func(action string, rule string, cliIP string)) {
	action := ActionAudit
	if stop {
		action = ActionBlock
	}
	logHitWithAction(action, phase, hostname, path, method, cli, reportFn)
}

func logHitWithAction(action, phase, hostname, path, method string, cli net.IP, reportFn func(action string, rule string, cliIP string)) {
	ipStr := "-"
	if cli != nil {
		ipStr = cli.String()
	}
	tag := "[waf audit]"
	switch action {
	case ActionBlock:
		tag = "[waf block]"
	case ActionPass:
		tag = "[waf pass]"
	}
	logger.Warnf("%s phase=%s client_ip=%s host=%s method=%s path=%s", tag, phase, ipStr, hostname, method, path)
	if reportFn != nil {
		reportFn(action, phase, ipStr)
	}
}

func matchesSignature(sr *sigRule, req *http.Request, pathOnly, rawQuery string) bool {
	hdrBuf := ""
	hdrBufFilled := false

	for ti, tk := range sr.targets {
		var blob string
		switch tk {
		case tkPath:
			blob = pathOnly
		case tkQuery:
			blob = rawQuery
		case tkURI:
			if rawQuery == "" {
				blob = pathOnly
			} else {
				blob = pathOnly + "?" + rawQuery
			}
		case tkHeaders:
			if !hdrBufFilled {
				hdrBuf = concatHeaders(req)
				hdrBufFilled = true
			}
			blob = hdrBuf
		case tkHeader:
			name := ""
			if ti < len(sr.hdrNames) {
				name = sr.hdrNames[ti]
			}
			blob = req.Header.Get(name)
		default:
			continue
		}
		if matchBlob(sr, blob) {
			return true
		}
	}
	return false
}

func concatHeaders(req *http.Request) string {
	if req == nil {
		return ""
	}
	var b strings.Builder
	for k, vv := range req.Header {
		kl := strings.ToLower(strings.TrimSpace(k))
		for _, v := range vv {
			b.WriteString(kl)
			b.WriteByte('=')
			b.WriteString(strings.TrimSpace(v))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func matchBlob(sr *sigRule, s string) bool {
	if sr.contains {
		return strings.Contains(s, sr.pattern)
	}
	if sr.re == nil {
		return false
	}
	return sr.re.MatchString(s)
}

func clientIP(r *http.Request, trust bool, idx int) net.IP {
	parseDirect := func() net.IP {
		hostPort := strings.TrimSpace(r.RemoteAddr)
		ipStr := hostPort
		if hop, _, err := net.SplitHostPort(hostPort); err == nil {
			ipStr = hop
		}
		return net.ParseIP(ipStr)
	}
	if !trust {
		return parseDirect()
	}

	xff := strings.TrimSpace(r.Header.Get(headerXForwardedFor))
	if xff != "" {
		var parts []net.IP
		for _, chunk := range strings.Split(xff, ",") {
			chunk = strings.TrimSpace(strings.Trim(chunk, `"`))
			if chunk == "" {
				continue
			}
			if ip := net.ParseIP(chunk); ip != nil {
				parts = append(parts, ip)
			}
		}
		if len(parts) == 0 {
			return parseDirect()
		}
		i := idx
		if i < 0 {
			i = len(parts) + i
		}
		if i >= 0 && i < len(parts) {
			return parts[i]
		}
		return parseDirect()
	}
	return parseDirect()
}
