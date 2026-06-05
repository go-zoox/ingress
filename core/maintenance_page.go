package core

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

const defaultMaintenanceTitle = "系统维护中"

func formatMaintenanceRecoveryTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func maintenancePageCopy(settings compiledMaintenanceSettings, window compiledMaintenanceWindow) (title, subtitle string) {
	title = strings.TrimSpace(settings.Title)
	subtitle = strings.TrimSpace(settings.Subtitle)
	if title == "" {
		title = defaultMaintenanceTitle
	}
	if subtitle == "" {
		if window.configured && !window.end.IsZero() {
			subtitle = fmt.Sprintf("我们正在进行计划维护，预计恢复时间：%s", formatMaintenanceRecoveryTime(window.end))
		} else {
			subtitle = "我们正在进行计划维护"
		}
	}
	return title, subtitle
}

func ingressMaintenancePageJSON(title, subtitle string) string {
	payload := ingressStatusBody{
		Status:   "maintenance",
		Title:    title,
		Subtitle: subtitle,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"status":"maintenance","title":"系统维护中","subtitle":"我们正在进行计划维护"}`
	}
	return string(b)
}

func ingressMaintenancePageHTML(title, subtitle string, exposeDetails bool, hostname, path, method, reason string) string {
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
		dlBlock = fmt.Sprintf(`<dl class="meta">
      <div><dt>Host</dt><dd>%s</dd></div>
      <div><dt>Path</dt><dd>%s</dd></div>
      <div><dt>Method</dt><dd>%s</dd></div>
    </dl>`, h, p, m)
	}

	footer := `如需帮助，请联系站点或服务管理员。`
	if exposeDetails {
		footer = `来自 ingress 网关的维护响应。下方详情仅供调试；对外部署请关闭 error_page_expose_details。`
	}
	footerBlock := fmt.Sprintf(`<footer class="foot">%s</footer>`, html.EscapeString(footer))

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<style>
:root {
  --bg: #050508;
  --card: rgba(12,14,20,.76);
  --ink: #eceff4;
  --muted: #8b93a7;
  --line: rgba(255,255,255,.09);
  --glow: rgba(251,191,36,.28);
}
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100svh;
  font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Microsoft YaHei", sans-serif;
  background: var(--bg);
  color: var(--ink);
}
.ambient {
  position: fixed; inset: 0; pointer-events: none;
  background:
    radial-gradient(900px 520px at 12%% -8%%, rgba(245,158,11,.1), transparent 58%%),
    radial-gradient(700px 420px at 88%% 8%%, rgba(251,191,36,.08), transparent 55%%),
    radial-gradient(600px 400px at 50%% 110%%, rgba(234,179,8,.05), transparent 60%%);
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
.tag {
  display: inline-block;
  font-size: .72rem; letter-spacing: .12em; text-transform: uppercase; color: #fbbf24;
  border: 1px solid rgba(251,191,36,.35); border-radius: 999px; padding: 4px 10px; margin-bottom: 18px;
}
.title {
  font-size: clamp(1.75rem, 5vw, 2.25rem); font-weight: 700; margin: 0 0 14px; letter-spacing: -.02em;
  text-shadow: 0 0 40px var(--glow);
}
.sub {
  font-size: .95rem; line-height: 1.75; color: var(--muted); margin: 0;
}
.reason {
  margin: 20px 0 0; padding: 12px 14px;
  background: rgba(251,191,36,.06); border: 1px solid rgba(255,255,255,.08); border-radius: 10px;
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
  font-size: .78rem; color: var(--muted); line-height: 1.6;
}
.bar {
  margin-top: 28px; height: 3px; border-radius: 999px; overflow: hidden; background: rgba(255,255,255,.06);
}
.bar span {
  display: block; height: 100%%; width: 42%%; border-radius: inherit;
  background: linear-gradient(90deg, rgba(251,191,36,.25), rgba(245,158,11,.75));
}
</style>
</head>
<body>
<div class="ambient"></div>
<main class="page">
  <article class="card">
    <span class="tag">维护</span>
    <h1 class="title">%s</h1>
    <p class="sub">%s</p>
    %s
    %s
    %s
    <div class="bar"><span></span></div>
  </article>
</main>
</body>
</html>`, title, title, subtitle, reasonBlock, dlBlock, footerBlock)
}
