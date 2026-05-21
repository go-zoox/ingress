(function () {
  const NAV = [
    { id: "overview", label: "总览", icon: "◉" },
    { id: "routes", label: "路由", icon: "⇄" },
    { id: "waf", label: "WAF", icon: "⛨" },
    { id: "tls", label: "TLS", icon: "🔒" },
    { id: "config", label: "配置", icon: "⌘" },
    { id: "logs", label: "日志", icon: "≡" },
  ];
  let savedYaml = MOCK.yamlBaseline.trim();
  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
  function toast(msg, type) {
    const el = document.createElement("div");
    el.className = "toast " + (type || "success");
    el.textContent = msg;
    $("#toasts").appendChild(el);
    setTimeout(function () { el.remove(); }, 4000);
  }
  function hostTypeBadge(t) {
    var c = t === "regex" ? "badge-regex" : t === "wildcard" ? "badge-wildcard" : "badge-exact";
    return '<span class="badge ' + c + '">' + t + "</span>";
  }
  function navigate(pageId) {
    $$(".nav button").forEach(function (b) { b.classList.toggle("active", b.dataset.page === pageId); });
    $$(".page").forEach(function (p) { p.classList.toggle("active", p.dataset.page === pageId); });
    location.hash = pageId;
  }
  function initNav() {
    var nav = $("#nav");
    NAV.forEach(function (item) {
      var btn = document.createElement("button");
      btn.type = "button";
      btn.dataset.page = item.id;
      btn.innerHTML = '<span class="icon">' + item.icon + "</span>" + item.label;
      btn.addEventListener("click", function () { navigate(item.id); });
      nav.appendChild(btn);
    });
    var hash = (location.hash || "#overview").slice(1);
    navigate(NAV.some(function (n) { return n.id === hash; }) ? hash : "overview");
    window.addEventListener("hashchange", function () {
      var h = (location.hash || "#overview").slice(1);
      if (NAV.some(function (n) { return n.id === h; })) navigate(h);
    });
    $$("[data-goto]").forEach(function (el) {
      el.addEventListener("click", function () { navigate(el.dataset.goto); });
    });
  }
  function card(cls, label, value, sub) {
    return '<div class="card ' + cls + '"><div class="label">' + label + '</div><div class="value">' + value + '</div><div class="sub">' + sub + '</div></div>';
  }
  function renderOverview() {
    var i = MOCK.instance;
    $("#sidebar-config-path").textContent = i.configPath;
    var certWarn = MOCK.certs.filter(function (c) { return c.status !== "ok"; }).length;
    var wafLabel = i.wafEnabled ? (i.wafLogOnly ? "审计" : "拦截") : "关";
    $("#overview-cards").innerHTML = [
      card("ok", "状态", "运行中", "PID " + i.pid + " · " + i.uptime),
      card("", "版本", i.version, "配置 hash " + i.configHash),
      card("", "监听", String(i.listenHTTP), "HTTPS " + i.listenHTTPS),
      card("", "路由规则", String(i.rulesCount), "上次 reload " + i.lastReload),
      card(i.wafLogOnly ? "warn" : "", "WAF", wafLabel, "log_only=" + i.wafLogOnly),
      card(certWarn ? "warn" : "ok", "证书", certWarn ? certWarn + " 需关注" : "正常", "TLS 证书有效期"),
    ].join("");
    var events = MOCK.wafEvents.slice(0, 4).map(function (e) {
      return "<tr><td>" + e.time + "</td><td><span class=\"badge badge-" + e.action + "\">" + e.action + "</span></td><td>" + e.rule + "</td><td>" + e.host + "</td><td>" + escapeHtml(e.path) + "</td></tr>";
    });
    $("#overview-events").innerHTML = "<table class=\"data\"><thead><tr><th>时间</th><th>动作</th><th>规则</th><th>Host</th><th>Path</th></tr></thead><tbody>" + events.join("") + "</tbody></table>";
  }
  function escapeHtml(s) {
    return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }
  function renderRoutes(filter) {
    var q = (filter || "").toLowerCase();
    var rows = MOCK.routes.filter(function (r) {
      if (!q) return true;
      return (r.host + " " + r.path + " " + r.target + " " + r.backendType).toLowerCase().indexOf(q) >= 0;
    }).map(function (r) {
      return "<tr><td><code>" + escapeHtml(r.host) + "</code></td><td>" + hostTypeBadge(r.hostType) + "</td><td>" + escapeHtml(r.path) + "</td><td>" + r.backendType + "</td><td><code>" + escapeHtml(r.target) + "</code></td></tr>";
    });
    $("#routes-table tbody").innerHTML = rows.join("") || '<tr><td colspan="5" class="empty-hint">无匹配规则</td></tr>';
  }
  function matchRoute(host, path) {
    host = (host || "").toLowerCase();
    path = path || "/";
    if (path.indexOf("/") !== 0) path = "/" + path;
    for (var i = 0; i < MOCK.routes.length; i++) {
      var r = MOCK.routes[i];
      if (r.hostType === "exact" && r.host.toLowerCase() !== host) continue;
      if (r.hostType === "wildcard") {
        var suffix = r.host.replace(/^\*\./, "");
        if (host.indexOf(suffix.toLowerCase(), host.length - suffix.length) === -1 && host !== suffix.toLowerCase()) continue;
      }
      if (r.hostType === "regex") {
        try { if (!new RegExp(r.host, "i").test(host)) continue; } catch (e) { continue; }
      }
      if (r.pathType === "exact" && r.path !== path) continue;
      if (r.pathType === "prefix" && r.path !== "/" && path.indexOf(r.path) !== 0) continue;
      return r;
    }
    return null;
  }
  function runMatch() {
    var host = $("#match-host").value.trim();
    var path = $("#match-path").value.trim();
    var hit = matchRoute(host, path);
    var box = $("#match-result");
    if (!hit) {
      box.className = "match-result miss";
      box.innerHTML = "<h3>未命中</h3><p>将走 fallback 或返回 404（原型模拟）。</p>";
      return;
    }
    box.className = "match-result hit";
    box.innerHTML = "<h3>命中规则 #" + hit.id + "</h3><dl><dt>Host</dt><dd>" + escapeHtml(hit.host) + " (" + hit.hostType + ")</dd><dt>Path</dt><dd>" + escapeHtml(hit.path) + "</dd><dt>Backend</dt><dd>" + hit.backendType + "</dd><dt>目标</dt><dd><code>" + escapeHtml(hit.target) + "</code></dd><dt>WAF</dt><dd>" + hit.waf + "</dd></dl>";
  }
  function renderWaf() {
    var i = MOCK.instance;
    $("#waf-cards").innerHTML = [
      card("", "状态", i.wafEnabled ? "已启用" : "关闭", "全局 waf.enabled"),
      card(i.wafLogOnly ? "warn" : "", "模式", i.wafLogOnly ? "仅审计" : "拦截", "log_only"),
      card("", "内置规则", "已加载", "builtin: true"),
    ].join("");
    renderWafTable($("#waf-filter").value);
  }
  function renderWafTable(filter) {
    var rows = MOCK.wafEvents.filter(function (e) { return filter === "all" || e.action === filter; }).map(function (e) {
      return "<tr><td>" + e.time + "</td><td><span class=\"badge badge-" + e.action + "\">" + e.action + "</span></td><td>" + e.rule + "</td><td>" + e.host + "</td><td><code>" + escapeHtml(e.path) + "</code></td><td>" + e.client + "</td></tr>";
    });
    $("#waf-table tbody").innerHTML = rows.join("");
  }
  function renderCerts() {
    var rows = MOCK.certs.map(function (c) {
      var status = '<span class="badge badge-exact">正常</span>';
      if (c.status === "warn") status = '<span class="badge badge-wildcard">即将过期</span>';
      if (c.status === "expired") status = '<span class="badge badge-block">已过期</span>';
      return "<tr><td>" + c.domain + "</td><td>" + c.issuer + "</td><td>" + c.notAfter + "</td><td>" + c.daysLeft + "</td><td>" + status + "</td></tr>";
    });
    $("#certs-table tbody").innerHTML = rows.join("");
  }
  function initConfig() {
    $("#yaml-editor").value = savedYaml;
    $("#yaml-editor").addEventListener("input", function () { $("#validate-output").innerHTML = ""; });
  }
  function validateYaml() {
    var text = $("#yaml-editor").value;
    var out = $("#validate-output");
    if (text.indexOf("version:") < 0) {
      out.innerHTML = '<p class="validate-err">缺少 version 字段（模拟校验）</p>';
      return false;
    }
    if (text.indexOf("ERROR_DEMO") >= 0) {
      out.innerHTML = '<p class="validate-err">rules[2].host: invalid regex（模拟）</p>';
      return false;
    }
    out.innerHTML = '<p class="validate-ok">✓ 校验通过（真实环境将调用 ingress validate）</p>';
    return true;
  }
  function showDiff() {
    var cur = $("#yaml-editor").value.split("\n");
    var base = savedYaml.split("\n");
    var lines = [];
    var max = Math.max(cur.length, base.length);
    for (var i = 0; i < max; i++) {
      var a = base[i]; var b = cur[i];
      if (a === b) { if (a !== undefined) lines.push("  " + a); }
      else {
        if (a !== undefined) lines.push('<span class="del">- ' + escapeHtml(a) + "</span>");
        if (b !== undefined) lines.push('<span class="add">+ ' + escapeHtml(b) + "</span>");
      }
    }
    $("#diff-content").innerHTML = lines.join("\n") || "(无变更)";
    $("#modal-diff").classList.add("open");
  }
  function openPublishModal() {
    $("#publish-path").textContent = MOCK.instance.configPath;
    $("#publish-status").textContent = "";
    $$("#publish-steps li").forEach(function (li) { li.classList.remove("done", "active"); });
    $("#modal-publish").classList.add("open");
  }
  function closeModals() {
    $$(".modal-overlay").forEach(function (m) { m.classList.remove("open"); });
  }
  function delay(ms) { return new Promise(function (r) { setTimeout(r, ms); }); }
  function runPublish() {
    var steps = $$("#publish-steps li");
    var status = $("#publish-status");
    var yaml = $("#yaml-editor").value;
    steps[0].classList.add("active");
    status.textContent = "正在校验…";
    return delay(600).then(function () {
      if (!validateYaml()) {
        status.textContent = "校验失败，已中止发布。";
        steps[0].classList.remove("active");
        return;
      }
      steps[0].classList.remove("active"); steps[0].classList.add("done");
      steps[1].classList.add("active");
      status.textContent = "正在写入 YAML…";
      return delay(800);
    }).then(function () {
      if (!steps[1].classList.contains("active") && !steps[1].classList.contains("done")) return;
      savedYaml = yaml.trim();
      steps[1].classList.remove("active"); steps[1].classList.add("done");
      steps[2].classList.add("active");
      status.textContent = "正在发送 SIGHUP…";
      return delay(700);
    }).then(function () {
      if (!steps[2].classList.contains("active") && !steps[2].classList.contains("done")) return;
      MOCK.instance.lastReload = new Date().toLocaleString("zh-CN", { hour12: false });
      MOCK.instance.lastReloadOK = true;
      MOCK.instance.configHash = Math.random().toString(16).slice(2, 10);
      steps[2].classList.remove("active"); steps[2].classList.add("done");
      status.textContent = "发布成功。";
      renderOverview();
      toast("配置已保存并 reload");
      setTimeout(closeModals, 900);
    });
  }
  function searchLogs() {
    var q = $("#log-q").value.toLowerCase();
    var host = $("#log-host").value.toLowerCase();
    var status = $("#log-status").value;
    var lines = MOCK.accessLogs.filter(function (line) {
      if (host && line.toLowerCase().indexOf(host) < 0) return false;
      if (q && line.toLowerCase().indexOf(q) < 0) return false;
      if (status) {
        var m = line.match(/"\s+(\d{3})\s/);
        if (!m || m[1].charAt(0) !== status) return false;
      }
      return true;
    });
    var container = $("#log-results");
    if (!lines.length) {
      container.innerHTML = '<div class="empty-hint">无匹配日志</div>';
      $("#log-count").textContent = "0 条";
      return;
    }
    container.innerHTML = lines.map(function (line) {
      var m = line.match(/"\s+(\d{3})\s/);
      var cls = m ? "status-" + m[1].charAt(0) + "xx" : "";
      return '<div class="log-line ' + cls + '">' + escapeHtml(line) + "</div>";
    }).join("");
    $("#log-count").textContent = lines.length + " 条";
  }
  function bindEvents() {
    $("#route-filter").addEventListener("input", function (e) { renderRoutes(e.target.value); });
    $("#btn-match").addEventListener("click", runMatch);
    $("#waf-filter").addEventListener("change", function (e) { renderWafTable(e.target.value); });
    $("#btn-validate").addEventListener("click", validateYaml);
    $("#btn-diff").addEventListener("click", showDiff);
    $("#btn-save").addEventListener("click", function () {
      if (!validateYaml()) { toast("请先修复校验错误", "error"); return; }
      savedYaml = $("#yaml-editor").value.trim();
      toast("已保存到 " + MOCK.instance.configPath);
    });
    $("#btn-publish").addEventListener("click", openPublishModal);
    $("#btn-publish-confirm").addEventListener("click", runPublish);
    $("#btn-quick-reload").addEventListener("click", function () { navigate("config"); openPublishModal(); });
    $$("[data-close-modal]").forEach(function (b) { b.addEventListener("click", closeModals); });
    $$(".modal-overlay").forEach(function (overlay) {
      overlay.addEventListener("click", function (e) { if (e.target === overlay) closeModals(); });
    });
    $("#btn-log-search").addEventListener("click", searchLogs);
    $("#btn-log-clear").addEventListener("click", function () {
      $("#log-q").value = ""; $("#log-host").value = ""; $("#log-status").value = ""; searchLogs();
    });
  }
  function init() {
    initNav(); renderOverview(); renderRoutes(); renderWaf(); renderCerts(); initConfig(); searchLogs(); bindEvents();
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
