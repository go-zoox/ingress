// Ingress HTML error pages for no matching route.
//
// Security: Public responses must not echo internal names, resolver errors, routing policy,
// or request metadata that helps attackers map the edge (see OWASP guidance on error handling).
// Full diagnostics belong in server logs only. Optional ErrorPageExposeDetails enables a
// verbose page for trusted/staging environments.
//
// Rationale: zoox renders HTTPError via ctx.HTML, but the message was plain text. We return
// a styled HTML document for route-miss responses; safe mode keeps copy generic.

package core

import (
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/go-zoox/zoox"
)

func matchErrorReason(err error) string {
	if errors.Is(err, ErrHostNotFound) {
		return "No rule matched this host and no global fallback is configured (or fallback is not in effect)."
	}
	return err.Error()
}

// writeIngressErrorPage renders an HTML error response. If exposeDetails is false, the body
// contains only a generic message (no host/path/method, no error strings, no product hints).
func writeIngressErrorPage(ctx *zoox.Context, status int, title, subtitle string, exposeDetails bool, hostname, path, method, reason string) {
	ctx.HTML(status, ingressErrorPageHTML(status, title, subtitle, exposeDetails, hostname, path, method, reason))
}

func ingressErrorPageHTML(status int, title, subtitle string, exposeDetails bool, hostname, path, method, reason string) string {
	code := strconv.Itoa(status)
	title = html.EscapeString(title)
	subtitle = html.EscapeString(subtitle)

	reasonBlock := ""
	dlBlock := ""
	if exposeDetails {
		reason = strings.TrimSpace(reason)
		if reason != "" {
			reasonBlock = fmt.Sprintf(`<p class="reason">%s</p>`, html.EscapeString(reason))
		}
		h := html.EscapeString(hostname)
		p := html.EscapeString(path)
		m := html.EscapeString(strings.ToUpper(method))
		dlBlock = fmt.Sprintf(`<dl>
      <div><dt>Host</dt><dd>%s</dd></div>
      <div><dt>Path</dt><dd>%s</dd></div>
      <div><dt>Method</dt><dd>%s</dd></div>
    </dl>`, h, p, m)
	}

	footer := `If you need help, contact the site or service administrator.`
	if exposeDetails {
		footer = `Response from the ingress gateway. Details below are for debugging; disable error_page_expose_details for public deployments.`
	}

	footer = html.EscapeString(footer)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s · %s</title>
<style>
:root { --fg:#0f172a; --muted:#64748b; --line:#e2e8f0; --accent:#2563eb; --bg:#f1f5f9; --card:#fff; }
@media (prefers-color-scheme: dark) {
  :root { --fg:#f8fafc; --muted:#94a3b8; --line:#334155; --accent:#60a5fa; --bg:#0f172a; --card:#1e293b; }
}
* { box-sizing: border-box; }
body { margin:0; min-height:100vh; font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: linear-gradient(160deg, var(--bg) 0%%, #e0e7ff 100%%); color: var(--fg); display:flex; align-items:center; justify-content:center; padding:24px; }
@media (prefers-color-scheme: dark) { body { background: linear-gradient(160deg, var(--bg) 0%%, #1e1b4b 100%%); } }
.wrap { max-width: 520px; width: 100%%; }
.card { background: var(--card); border: 1px solid var(--line); border-radius: 16px; padding: 32px 28px; box-shadow: 0 10px 40px rgba(15,23,42,.08); }
.badge { display:inline-block; font-weight:700; font-size: 2.5rem; line-height:1; letter-spacing:-0.02em; color: var(--accent); margin-bottom: 8px; }
h1 { font-size: 1.35rem; font-weight: 600; margin: 0 0 8px; line-height: 1.3; }
.sub { margin: 0 0 20px; color: var(--muted); font-size: 0.95rem; line-height: 1.5; }
.reason { margin: 0 0 20px; padding: 12px 14px; background: rgba(37,99,235,.06); border-radius: 10px; border-left: 3px solid var(--accent); font-size: 0.9rem; line-height: 1.5; color: var(--fg); }
dl { margin: 0; display: grid; gap: 10px; font-size: 0.875rem; }
dt { color: var(--muted); font-weight: 500; margin: 0; }
dd { margin: 2px 0 0; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 0.82rem; word-break: break-all; }
footer { margin-top: 24px; padding-top: 16px; border-top: 1px solid var(--line); font-size: 0.75rem; color: var(--muted); line-height: 1.5; }
</style>
</head>
<body>
<div class="wrap">
  <div class="card">
    <div class="badge">%s</div>
    <h1>%s</h1>
    <p class="sub">%s</p>
    %s
    %s
    <footer>%s</footer>
  </div>
</div>
</body>
</html>`, title, code, code, title, subtitle, reasonBlock, dlBlock, footer)
}
