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
	ctx.HTML(status, ingressErrorPageHTML(status, title, subtitle, exposeDetails, hostname, path, method, reason, ""))
}

func ingressErrorPageHTML(status int, title, subtitle string, exposeDetails bool, hostname, path, method, reason, categoryTag string) string {
	code := strconv.Itoa(status)
	title = html.EscapeString(title)
	subtitle = html.EscapeString(subtitle)
	tag := strings.TrimSpace(categoryTag)
	if tag == "" {
		tag = errorPageCategoryTag(status)
	}

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
		dlBlock = fmt.Sprintf(`<dl class="meta">
      <div><dt>Host</dt><dd>%s</dd></div>
      <div><dt>Path</dt><dd>%s</dd></div>
      <div><dt>Method</dt><dd>%s</dd></div>
    </dl>`, h, p, m)
	}

	footer := `If you need help, contact the site or service administrator.`
	if exposeDetails {
		footer = `Response from the ingress gateway. Details below are for debugging; disable error_page_expose_details for public deployments.`
	}

	footerBlock := fmt.Sprintf(`<footer class="foot">%s</footer>`, html.EscapeString(footer))

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s · %s</title>
<style>
:root {
  --bg: #050508;
  --card: rgba(12,14,20,.76);
  --ink: #eceff4;
  --muted: #8b93a7;
  --line: rgba(255,255,255,.09);
  --glow: rgba(147,197,253,.35);
}
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100svh;
  font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: var(--bg);
  color: var(--ink);
}
.ambient {
  position: fixed; inset: 0; pointer-events: none;
  background:
    radial-gradient(900px 520px at 12%% -8%%, rgba(59,130,246,.12), transparent 58%%),
    radial-gradient(700px 420px at 88%% 8%%, rgba(167,139,250,.1), transparent 55%%),
    radial-gradient(600px 400px at 50%% 110%%, rgba(56,189,248,.06), transparent 60%%);
}
.noise {
  position: fixed; inset: 0; pointer-events: none; opacity: .035;
  background-image: url("data:image/svg+xml,%%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%%3E%%3Cfilter id='n'%%3E%%3CfeTurbulence type='fractalNoise' baseFrequency='.85' numOctaves='4' stitchTiles='stitch'/%%3E%%3C/filter%%3E%%3Crect width='100%%25' height='100%%25' filter='url(%%23n)'/%%3E%%3C/svg%%3E");
}
.page {
  position: relative; z-index: 1;
  min-height: 100svh; display: grid; place-items: center; padding: 56px 24px;
}
.card {
  width: min(520px, 100%%); padding: 36px 32px 30px; border-radius: 16px;
  background: var(--card); border: 1px solid var(--line);
  box-shadow: 0 0 0 1px rgba(255,255,255,.02) inset, 0 40px 100px rgba(0,0,0,.55);
  backdrop-filter: blur(24px);
  position: relative; overflow: hidden;
}
.card::after {
  content: ""; position: absolute; top: 0; left: 32px; right: 32px; height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,.22), transparent);
}
.row { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; margin-bottom: 22px; }
.tag {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: .68rem; letter-spacing: .08em; text-transform: uppercase; color: var(--muted);
}
.code {
  font-size: clamp(3rem, 10vw, 4.2rem); font-weight: 700; line-height: 1; letter-spacing: -.05em; margin: 0;
  text-shadow: 0 0 40px var(--glow);
}
.title { font-size: 1.25rem; font-weight: 600; margin: 0 0 10px; letter-spacing: -.01em; }
.sub {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: .84rem; line-height: 1.7; color: var(--muted); margin: 0;
}
.reason {
  margin: 20px 0 0; padding: 12px 14px;
  background: rgba(147,197,253,.06); border: 1px solid rgba(255,255,255,.08); border-radius: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: .8rem; line-height: 1.6; color: var(--ink);
}
.meta { margin: 20px 0 0; display: grid; gap: 10px; font-size: .78rem; }
.meta dt {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--muted); text-transform: uppercase; letter-spacing: .06em; font-size: .68rem; margin: 0;
}
.meta dd {
  margin: 4px 0 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: .82rem; word-break: break-all; color: var(--ink);
}
.foot {
  margin-top: 24px; padding-top: 16px; border-top: 1px solid var(--line);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: .72rem; color: var(--muted); line-height: 1.6;
}
.bar {
  margin-top: 28px; height: 3px; border-radius: 999px; overflow: hidden; background: rgba(255,255,255,.06);
}
.bar span {
  display: block; height: 100%%; width: 38%%; border-radius: inherit;
  background: linear-gradient(90deg, rgba(147,197,253,.2), rgba(167,139,250,.75));
}
</style>
</head>
<body>
<div class="ambient"></div>
<div class="noise"></div>
<main class="page">
  <article class="card">
    <div class="row"><span class="tag">%s</span><p class="code">%s</p></div>
    <h1 class="title">%s</h1>
    <p class="sub">%s</p>
    %s
    %s
    %s
    <div class="bar"><span></span></div>
  </article>
</main>
</body>
</html>`, title, code, tag, code, title, subtitle, reasonBlock, dlBlock, footerBlock)
}

func errorPageCategoryTag(status int) string {
	switch status {
	case 401, 403, 404:
		return "client"
	case 500:
		return "origin"
	default:
		return "upstream"
	}
}
