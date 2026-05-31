package core

import (
	"net/http"
	"time"

	"github.com/go-zoox/ingress/core/security"
	"github.com/go-zoox/zoox"
)

func (c *core) securityForRule(ruleIdx int) *security.Profile {
	if c.security == nil {
		return nil
	}
	if ruleIdx >= 0 && ruleIdx < len(c.security.ByRule) {
		return c.security.ByRule[ruleIdx]
	}
	return c.security.Global
}

func (c *core) securityForMatch(ruleIdx, pathIdx int) *security.Profile {
	if c.security == nil {
		return nil
	}
	if ruleIdx >= 0 && pathIdx >= 0 &&
		ruleIdx < len(c.security.ByPath) &&
		pathIdx < len(c.security.ByPath[ruleIdx]) {
		if p := c.security.ByPath[ruleIdx][pathIdx]; p != nil {
			return p
		}
	}
	return c.securityForRule(ruleIdx)
}

func (c *core) securityGlobal() *security.Profile {
	if c.security == nil {
		return nil
	}
	return c.security.Global
}

func applySecurityHeaders(ctx *zoox.Context, prof *security.Profile) {
	if prof == nil || !prof.Active {
		return
	}
	security.ApplyHeaders(ctx.Writer.Header(), prof, ctx.Request)
}

func (c *core) handleSecurityPreflight(ctx *zoox.Context, prof *security.Profile, hostname, target, method, path, proto string, reqStart time.Time) bool {
	if prof == nil || !prof.Active {
		return false
	}
	if !security.HandlePreflight(ctx.Writer, ctx.Request, prof) {
		return false
	}
	c.logAccess(ctx, hostname, target, method, path, proto, http.StatusNoContent, time.Since(reqStart), accessLogMeta{
		UpstreamStatus:         http.StatusNoContent,
		UpstreamResponseLength: 0,
	})
	return true
}
