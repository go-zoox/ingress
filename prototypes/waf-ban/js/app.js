/* WAF 封禁原型 — 交互逻辑（全部本地模拟，无后端请求） */

(function () {
  "use strict";

  const $ = (sel, root) => (root || document).querySelector(sel);
  const $$ = (sel, root) => Array.prototype.slice.call((root || document).querySelectorAll(sel));

  const state = {
    page: "events",
    banTab: "active",
    selectedEvents: new Set(),
    bans: MOCK.bans.slice(),
    expired: MOCK.expiredBans.slice(),
    audit: MOCK.audit.slice(),
    activeBanId: null,
    banDetailEl: null,
  };

  /* 演示用的 allow 白名单：命中时给出"封禁仍生效"的提示 */
  const ALLOW_LIST = ["192.0.2.155"];
  /* 演示用的已知受限地址 */
  const MOCK_TRUSTED = MOCK.trustedProxies;

  /* ---------------- 通用工具 ---------------- */

  function escapeHtml(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function toast(msg, type) {
    const el = document.createElement("div");
    el.className = "toast " + (type === "error" ? "error" : "success");
    el.textContent = msg;
    $("#toasts").appendChild(el);
    setTimeout(() => el.remove(), 3600);
  }

  function open(el) {
    el.style.display = "flex";
  }

  function close(el) {
    el.style.display = "none";
  }

  /* ---------------- 侧栏导航 ---------------- */

  const NAV_GROUPS = [
    { label: "监控", items: [{ id: "overview", icon: "◉", label: "总览" }, { id: "events-waf", icon: "◈", label: "事件" }] },
    { label: "流量", items: [{ id: "routes", icon: "⇄", label: "路由" }, { id: "services", icon: "▤", label: "服务" }] },
    {
      label: "安全",
      items: [
        { id: "events", icon: "⛨", label: "WAF 事件", active: true },
        { id: "bans", icon: "⊘", label: "封禁名单", active: true, badge: "18" },
        { id: "tls", icon: "🔒", label: "TLS" },
      ],
    },
    { label: "维护", items: [{ id: "maintenance", icon: "⚑", label: "维护模式" }, { id: "jobs", icon: "◷", label: "定时任务" }] },
    { label: "系统", items: [{ id: "config", icon: "⌘", label: "配置" }, { id: "settings", icon: "◎", label: "设置" }] },
  ];

  function renderNav() {
    const nav = $("#nav");
    nav.innerHTML = NAV_GROUPS.map((g) => {
      const items = g.items
        .map((it) => {
          const badge = it.badge ? '<span class="nav-badge">' + escapeHtml(it.badge) + "</span>" : "";
          const cls = it.id === state.page ? "active" : "";
          const dim = it.active ? "" : ' style="opacity:.45"';
          return (
            '<a href="#' +
            it.id +
            '" class="' +
            cls +
            '"' +
            dim +
            ' data-nav="' +
            it.id +
            '"><span class="icon">' +
            it.icon +
            '</span><span class="nav-label">' +
            escapeHtml(it.label) +
            "</span>" +
            badge +
            "</a>"
          );
        })
        .join("");
      return '<div class="nav-group"><div class="nav-group-label">' + g.label + "</div>" + items + "</div>";
    }).join("");

    $$("[data-nav]", nav).forEach((a) => {
      a.addEventListener("click", (e) => {
        e.preventDefault();
        const id = a.dataset.nav;
        if (id === "events" || id === "bans") {
          navigate(id);
        } else {
          toast("本原型仅演示 WAF 封禁相关页面：" + a.textContent.trim(), "error");
        }
      });
    });
  }

  function navigate(page) {
    state.page = page;
    $$(".nav a").forEach((a) => a.classList.toggle("active", a.dataset.nav === page));
    $$(".page").forEach((p) => p.classList.toggle("active", p.dataset.page === page));
    if (location.hash !== "#" + page) location.hash = page;
  }

  /* ---------------- 事件页 ---------------- */

  const RULE_LABEL = {
    "sql-injection-uri": "SQL 注入（URI）",
    "path-traversal": "路径遍历",
    "scanner-ua": "扫描器 UA",
    "xss-uri": "XSS（URI）",
    "jndi-lookup": "JNDI 注入",
    "crlf-injection": "CRLF 注入（CRLF）",
    "ip deny": "IP 黑名单",
  };

  function renderEvents() {
    const body = $("#events-body");
    const rows = MOCK.events
      .filter((e) => {
        const a = $("#f-action").value;
        const h = $("#f-host").value;
        const ip = $("#f-ip").value.trim();
        const rule = $("#f-rule").value.trim();
        if (a !== "all" && e.action !== a) return false;
        if (h && e.host !== h) return false;
        if (ip && e.ip.indexOf(ip) < 0) return false;
        if (rule && e.rule.indexOf(rule) < 0) return false;
        return true;
      })
      .map((e) => {
        const checked = state.selectedEvents.has(e.id) ? " checked" : "";
        const ruleText = RULE_LABEL[e.rule] || e.rule;
        const uaTarget = e.ua ? '<button type="button" class="btn btn-sm btn-ghost" data-ban-ua="' + e.id + '">封禁 UA</button>' : "";
        return (
          "<tr>" +
          '<td><input type="checkbox" class="ev-check" data-id="' + e.id + '"' + checked + " /></td>" +
          '<td class="col-time">' + e.time + "</td>" +
          '<td class="col-action"><span class="badge badge-' + e.action + '">' + e.action + "</span></td>" +
          '<td class="col-rule">' + escapeHtml(ruleText) + "</td>" +
          '<td class="col-host">' + escapeHtml(e.host) + "</td>" +
          '<td class="col-path"><code>' + escapeHtml(e.path) + "</code></td>" +
          '<td class="col-ip">' + escapeHtml(e.ip) + "</td>" +
          '<td class="col-actions">' +
          '<button type="button" class="btn btn-sm btn-ghost" data-detail="' + e.id + '">详情</button>' +
          uaTarget +
          '<button type="button" class="btn btn-sm btn-danger" data-ban-ip="' + e.id + '">封禁 IP</button>' +
          "</td>" +
          "</tr>"
        );
      });

    body.innerHTML = rows.length
      ? rows.join("")
      : '<tr><td colspan="8"><div class="empty">当前筛选条件下暂无事件</div></td></tr>';

    $("#events-count").textContent = "（" + rows.length + " 条）";

    $$(".ev-check", body).forEach((cb) => {
      cb.addEventListener("change", () => {
        const id = Number(cb.dataset.id);
        if (cb.checked) state.selectedEvents.add(id);
        else state.selectedEvents.delete(id);
        syncBulk();
      });
    });
    $$("[data-ban-ip]", body).forEach((b) =>
      b.addEventListener("click", () => openFromEvent(Number(b.dataset.banIp), "ip"))
    );
    $$("[data-ban-ua]", body).forEach((b) =>
      b.addEventListener("click", () => openFromEvent(Number(b.dataset.banUa), "ua"))
    );
    $$("[data-detail]", body).forEach((b) =>
      b.addEventListener("click", () => toast("原型未实现事件详情抽屉（现有后台已有）", "error"))
    );

    state.selectedEvents = new Set(
      Array.from(state.selectedEvents).filter((id) => MOCK.events.some((e) => e.id === id))
    );
    syncBulk();
  }

  function syncBulk() {
    const n = state.selectedEvents.size;
    $("#bulk-sel").textContent = "已选 " + n + " 项";
    $("#btn-bulk-ban").disabled = n === 0;
    const all = $("#events-check-all");
    all.checked = n > 0 && n === visibleEventCount();
  }

  function visibleEventCount() {
    return $$("#events-body tr").filter((tr) => $(".ev-check", tr)).length;
  }

  /* ---------------- 从事件打开封禁抽屉 ---------------- */

  const drawerFromEvent = () => $("#drawer-from-event");

  function openFromEvent(eventId, kind) {
    const e = MOCK.events.find((x) => x.id === eventId);
    if (!e) return;

    $("#fe-title").textContent = kind === "ip" ? "封禁 IP" : "封禁 UA";
    $("#fe-kind").textContent = kind === "ip" ? "IP" : "UA";
    $("#fe-value").textContent = kind === "ip" ? e.ip : e.ua;
    $("#fe-meta").textContent =
      "来自事件 #" + e.id + " · " + (RULE_LABEL[e.rule] || e.rule) + " · " + e.host;

    $("#fe-hits").textContent = String(e.hits24h);
    $("#fe-hosts").textContent = String(e.hosts);
    $("#fe-uas").textContent = String(e.uas);
    $("#fe-paths").textContent = String(e.paths);

    /* 影响面提示 */
    const notices = [];
    if (kind === "ip" && isTrusted(e.ip)) {
      notices.push(
        notice(
          "danger",
          "目标命中 trusted_proxies，应拒绝封禁",
          "该地址是反向代理/负载均衡自身，封禁会导致整站不可访问。"
        )
      );
    } else if (kind === "ip" && e.uas >= 20) {
      notices.push(
        notice(
          "danger",
          "该 IP 疑似 NAT / 运营商出口",
          "近 24h 出现 " + e.uas + " 个不同 UA，封禁可能误伤大量真实用户。建议缩短时长或改为按 UA 封禁。"
        )
      );
    } else if (kind === "ip") {
      notices.push(
        notice(
          "warn",
          "该 IP 的 UA 单一、路径多样",
          "疑似扫描器/自动化工具，封禁误伤风险较低。"
        )
      );
    } else {
      notices.push(
        notice(
          "info",
          "UA 是客户端可控字段",
          "攻击者更换 UA 即可绕过，按 UA 封禁只适合挡低级爬虫，属于止血手段。"
        )
      );
    }
    if (kind === "ip" && ALLOW_LIST.indexOf(e.ip) >= 0) {
      notices.push(
        notice("info", "该 IP 在 allow 白名单中", "运行时封禁优先于 allow 白名单，依然会生效。")
      );
    }
    $("#fe-notices").innerHTML = notices.join("");

    $("#fe-reason").value =
      kind === "ip" ? "疑似扫描：" + (RULE_LABEL[e.rule] || e.rule) : "按 UA 封禁：" + e.ua.slice(0, 40);

    const submit = $("#fe-submit");
    const blocked = kind === "ip" && isTrusted(e.ip);
    submit.disabled = blocked;
    submit.textContent = blocked ? "无法封禁（受限地址）" : "确认封禁";
    submit.dataset.target = kind === "ip" ? e.ip : e.ua;
    submit.dataset.kind = kind;
    submit.dataset.eventId = String(e.id);

    resetTtl($("#fe-ttl"), "24h");
    resetScope(drawerFromEvent());
    open(drawerFromEvent());
  }

  function isTrusted(ip) {
    return MOCK_TRUSTED.some((cidr) => MOCK.ipInCidr(ip, cidr));
  }

  function notice(level, title, body) {
    const icon = { danger: "⚠", warn: "⚠", info: "i", muted: "i" }[level] || "i";
    return (
      '<div class="notice notice-' +
      level +
      '"><span class="n-icon">' +
      icon +
      '</span><span><span class="n-title">' +
      escapeHtml(title) +
      "</span>" +
      body +
      "</span></div>"
    );
  }

  /* ---------------- 封禁名单页 ---------------- */

  const KIND_LABEL = { ip: "IP", cidr: "CIDR", ua: "UA" };

  function renderBans() {
    const q = ($("#ban-search").value || "").trim().toLowerCase();
    const kindFilter = $("#ban-kind-filter").value;
    const rows = state.bans
      .filter((b) => {
        if (kindFilter !== "all" && b.kind !== kindFilter) return false;
        if (!q) return true;
        return (b.value + " " + b.reason + " " + b.actor).toLowerCase().indexOf(q) >= 0;
      })
      .map((b) => {
        const expiring = /剩余\\s*\\d+m/.test(b.ttlLabel) || /剩余\\s*[1-9]h/.test(b.ttlLabel);
        const scopeCell =
          b.scope === "全局"
            ? '<span class="badge badge-muted">全局</span>'
            : '<span class="badge badge-warn">指定 Host</span><div class="sub-line">' +
              escapeHtml(b.scopeHosts.join(", ")) +
              "</div>";
        return (
          "<tr>" +
          '<td><span class="badge badge-kind">' + KIND_LABEL[b.kind] + "</span></td>" +
          '<td class="ban-value-cell">' +
          escapeHtml(b.value) +
          '<div class="sub-line">' +
          escapeHtml(b.reason) +
          (b.kind === "ua" ? " · " + b.ua : "") +
          "</div></td>" +
          "<td>" + scopeCell + "</td>" +
          '<td class="' + (expiring ? "text-warn" : "") + '">' + escapeHtml(b.expiresAt) +
          '<div class="sub-line">' + escapeHtml(b.ttlLabel) + "</div></td>" +
          '<td class="mono">' + b.hits.toLocaleString() + "</td>" +
          '<td class="col-time">' + escapeHtml(b.actor) + "</td>" +
          '<td class="row-actions">' +
          '<button type="button" class="btn btn-sm btn-ghost" data-ban-view="' + b.id + '">详情</button>' +
          '<button type="button" class="btn btn-sm btn-ghost" data-ban-renew="' + b.id + '">续期</button>' +
          '<button type="button" class="btn btn-sm btn-danger" data-ban-remove="' + b.id + '">解封</button>' +
          "</td>" +
          "</tr>"
        );
      });

    $("#bans-body").innerHTML = rows.length
      ? rows.join("")
      : '<tr><td colspan="7"><div class="empty">没有匹配的封禁条目</div></td></tr>';

    $$("#bans-body [data-ban-view]").forEach((b) =>
      b.addEventListener("click", () => showBanDetail(Number(b.dataset.banView)))
    );
    $$("#bans-body [data-ban-renew]").forEach((b) =>
      b.addEventListener("click", () => renewBan(Number(b.dataset.banRenew)))
    );
    $$("#bans-body [data-ban-remove]").forEach((b) =>
      b.addEventListener("click", () => removeBan(Number(b.dataset.banRemove)))
    );

    /* 过期列表 */
    $("#bans-expired-body").innerHTML = state.expired
      .map(
        (b) =>
          "<tr>" +
          '<td><span class="badge badge-kind">' + KIND_LABEL[b.kind] + "</span></td>" +
          '<td class="ban-value-cell">' + escapeHtml(b.value) +
          '<div class="sub-line">' + escapeHtml(b.reason) + "</div></td>" +
          '<td class="col-time">' + escapeHtml(b.expiredAt) + "</td>" +
          '<td class="mono">' + b.hits.toLocaleString() + "</td>" +
          '<td class="col-time">' + escapeHtml(b.actor) + "</td>" +
          '<td class="row-actions"><button type="button" class="btn btn-sm" data-ban-again="' +
          b.id + '">重新封禁</button></td>' +
          "</tr>"
      )
      .join("");

    $$("#bans-expired-body [data-ban-again]").forEach((btn) =>
      btn.addEventListener("click", () => {
        openBanCreate();
        toast("已带入上次封禁目标，请确认时长");
      })
    );

    /* 审计列表 */
    $("#bans-audit-body").innerHTML = state.audit
      .map(
        (a) =>
          "<tr>" +
          '<td class="col-time">' + escapeHtml(a.time) + "</td>" +
          '<td class="mono">' + escapeHtml(a.action) + "</td>" +
          "<td>" + escapeHtml(a.detail) + "</td>" +
          '<td class="col-time">' + escapeHtml(a.actor) + "</td>" +
          "</tr>"
      )
      .join("");

    refreshCounts();
  }

  function refreshCounts() {
    const n = state.bans.length;
    $("#chrome-count").textContent = String(n);
    $("#events-card-bans").textContent = String(n);

    const ipN = state.bans.filter((b) => b.kind !== "ua").length;
    const uaN = state.bans.filter((b) => b.kind === "ua").length;
    $("#ban-card-active").textContent = String(n);
    $("#ban-card-split").textContent = "IP " + ipN + " · UA " + uaN;

    /* "即将到期"：ttlLabel 形如「剩余 48m」或「剩余 Nh」且 N<3 */
    const expiring = state.bans.filter((b) => {
      const m = /剩余\s*(\d+)\s*([mhd])/.exec(b.ttlLabel);
      if (!m) return false;
      const v = Number(m[1]);
      if (m[2] === "m") return true;
      return m[2] === "h" && v <= 2;
    }).length;
    $("#ban-card-expiring").textContent = String(expiring);
    $("#ban-card-expired").textContent = String(state.expired.length);

    $("#tab-count-active").textContent = String(n);
    $("#tab-count-expired").textContent = String(state.expired.length);

    const badge = $('.nav a[data-nav="bans"] .nav-badge');
    if (badge) badge.textContent = String(n);
  }

  function showBanDetail(id) {
    const b = state.bans.find((x) => x.id === id);
    if (!b) return;
    state.activeBanId = id;
    $("#bd-title").textContent = "封禁详情 #" + b.id;
    $("#bd-body").innerHTML =
      '<p class="section-title">封禁目标</p>' +
      '<div class="target-card">' +
      '<div class="target-row"><span class="badge badge-kind">' + KIND_LABEL[b.kind] + "</span>" +
      '<span class="target-value">' + escapeHtml(b.value) + "</span></div>" +
      '<div class="target-meta">' + escapeHtml(b.reason) + "</div></div>" +
      '<p class="section-title">状态</p>' +
      '<div class="metric-grid">' +
      metric("命中次数", b.hits.toLocaleString(), "") +
      metric("最近命中", b.lastHit, "") +
      metric("到期", b.expiresAt, b.ttlLabel.indexOf("剩余") >= 0 ? "warn" : "") +
      metric("生效范围", b.scope, "") +
      "</div>" +
      '<p class="section-title">来源</p>' +
      '<div class="target-card"><table class="data" style="font-size:12px">' +
      row("创建时间", b.createdAt) +
      row("操作人", b.actor) +
      row("来源", b.source === "manual" ? "手动封禁" : "自动封禁") +
      row("匹配方式", b.kind === "ua" ? (b.ua === "regex" ? "正则" : "包含子串") : "精确匹配 / CIDR") +
      row("作用域", b.scope === "全局" ? "全部 Host" : b.scopeHosts.join(", ")) +
      "</table></div>" +
      '<div class="notice notice-muted" style="margin-top:14px"><span class="n-icon">i</span>' +
      "<span>该封禁独立于 <code>waf.enabled</code> 与配置 reload，WAF 总开关关闭时仍然生效。</span></div>";
    open($("#drawer-ban-detail"));
  }

  function metric(label, value, cls) {
    return (
      '<div class="metric"><div class="m-label">' + escapeHtml(label) + "</div>" +
      '<div class="m-value ' + (cls || "") + '">' + escapeHtml(value) + "</div></div>"
    );
  }

  function row(k, v) {
    return "<tr><td style=\"color:var(--text-muted);width:90px\">" + escapeHtml(k) + "</td><td>" + escapeHtml(v) + "</td></tr>";
  }

  function removeBan(id) {
    state.bans = state.bans.filter((b) => b.id !== id);
    state.audit.unshift({
      time: nowHHMM(),
      action: "waf.ban.remove",
      detail: "提前解封 ban#" + id,
      actor: "admin",
    });
    close($("#drawer-ban-detail"));
    renderBans();
    toast("已解封，立即生效");
  }

  function renewBan(id) {
    const b = state.bans.find((x) => x.id === id);
    if (!b) return;
    b.ttlLabel = "剩余 24h";
    b.expiresAt = "11-08 11:30";
    state.audit.unshift({
      time: nowHHMM(),
      action: "waf.ban.renew",
      detail: "续期 ban#" + id + " 至 " + b.expiresAt,
      actor: "admin",
    });
    renderBans();
    close($("#drawer-ban-detail"));
    toast("已续期 24 小时");
  }

  function nowHHMM() {
    const d = new Date();
    return String(d.getHours()).padStart(2, "0") + ":" + String(d.getMinutes()).padStart(2, "0");
  }

  /* ---------------- 新增封禁抽屉 ---------------- */

  const drawerBanCreate = () => $("#drawer-ban-create");
  let createKind = "ip";

  function openBanCreate() {
    createKind = "ip";
    $$("#bc-kind .seg").forEach((s) => s.classList.toggle("active", s.dataset.kind === "ip"));
    applyCreateKind();
    $("#bc-value").value = "";
    $("#bc-reason").value = "";
    $("#bc-value-field").classList.remove("invalid");
    $("#bc-trusted-warn").style.display = "none";
    $("#bc-nat-warn").style.display = "none";
    $("#bc-allow-warn").style.display = "none";
    $("#bc-submit").disabled = false;
    resetTtl($("#bc-ttl"), "24h");
    resetScope(drawerBanCreate());
    open(drawerBanCreate());
    setTimeout(() => $("#bc-value").focus(), 60);
  }

  function applyCreateKind() {
    const ipMode = createKind === "ip";
    $("#bc-value-label").textContent = ipMode ? "IP 或 CIDR" : "User-Agent 关键词";
    $("#bc-value").placeholder = ipMode
      ? "203.0.113.42 或 203.0.113.0/24"
      : "例：sqlmap 或 (?i)(nikto|masscan)";
    $("#bc-value-hint").textContent = ipMode
      ? "支持单个 IP 或 CIDR 网段；IPv6 自动归一化（::ffff:1.2.3.4 → 1.2.3.4）。"
      : "按子串或正则匹配 User-Agent 请求头，大小写不敏感。";
    $("#bc-ua-mode-field").style.display = ipMode ? "none" : "block";
    $("#bc-trusted-warn").style.display = "none";
    $("#bc-nat-warn").style.display = "none";
    $("#bc-allow-warn").style.display = "none";
    $("#bc-submit").disabled = false;
    $("#bc-value-field").classList.remove("invalid");
  }

  const IPV4 = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
  const CIDR4 = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/;
  const IPV6 = /^[0-9a-fA-F:]+$/;

  function validIPValue(v) {
    if (CIDR4.test(v)) {
      const bits = parseInt(v.split("/")[1], 10);
      if (bits < 0 || bits > 32) return false;
      return IPV4.test(v.split("/")[0]);
    }
    if (IPV4.test(v)) return v.split(".").every((o) => Number(o) >= 0 && Number(o) <= 255);
    if (v.indexOf(":") >= 0 && IPV6.test(v)) return true;
    return false;
  }

  function validateCreateValue() {
    const v = $("#bc-value").value.trim();
    const field = $("#bc-value-field");
    const submit = $("#bc-submit");
    $("#bc-trusted-warn").style.display = "none";
    $("#bc-nat-warn").style.display = "none";
    $("#bc-allow-warn").style.display = "none";

    if (!v) {
      field.classList.remove("invalid");
      submit.disabled = false;
      return;
    }

    if (createKind === "ip") {
      if (!validIPValue(v)) {
        field.classList.add("invalid");
        submit.disabled = false;
        return;
      }
      field.classList.remove("invalid");

      if (isTrusted(v.split("/")[0])) {
        $("#bc-trusted-warn").style.display = "flex";
        submit.disabled = true;
        return;
      }
      submit.disabled = false;

      const known = MOCK.events.filter((e) => e.ip === v);
      if (known.length && known[0].uas >= 20) {
        $("#bc-nat-warn").style.display = "flex";
      }
      if (ALLOW_LIST.indexOf(v) >= 0) {
        $("#bc-allow-warn").style.display = "flex";
      }
    } else {
      field.classList.remove("invalid");
      submit.disabled = false;
    }
  }

  /* ---------------- 批量封禁 ---------------- */

  function openBulk() {
    const ids = Array.from(state.selectedEvents);
    const ips = [];
    ids.forEach((id) => {
      const e = MOCK.events.find((x) => x.id === id);
      if (e && ips.indexOf(e.ip) < 0) ips.push(e.ip);
    });
    $("#bulk-count").textContent = String(ips.length);
    $("#bulk-list").innerHTML = ips
      .map((ip) => {
        const trusted = isTrusted(ip) ? ' <span class="badge badge-danger">受限地址，将跳过</span>' : "";
        return "<div>• " + escapeHtml(ip) + trusted + "</div>";
      })
      .join("");
    resetTtl($("#bulk-ttl"), "24h");
    open($("#drawer-bulk"));
  }

  /* ---------------- 控件通用行为 ---------------- */

  function resetTtl(container, value) {
    $$(".seg", container).forEach((s) => s.classList.toggle("active", s.dataset.ttl === value));
    container.dataset.value = value;
    const warn = $("#fe-ttl-warn");
    if (container.id === "fe-ttl" && warn) warn.style.display = "none";
  }

  function resetScope(root) {
    $$('input[type="radio"][value="global"]', root).forEach((r) => (r.checked = true));
    $$(".host-chips", root).forEach((el) => (el.style.display = "none"));
  }

  function bindSegmented(container) {
    $$(".seg", container).forEach((seg) => {
      seg.addEventListener("click", () => {
        $$(".seg", container).forEach((s) => s.classList.remove("active"));
        seg.classList.add("active");
        container.dataset.value = seg.dataset.ttl;
        if (container.id === "fe-ttl") {
          $("#fe-ttl-warn").style.display = seg.dataset.ttl === "forever" ? "flex" : "none";
          $("#fe-submit").textContent =
            seg.dataset.ttl === "forever" ? "永久封禁（再次确认）" : "确认封禁";
        }
      });
    });
  }

  function bindScope(root) {
    $$('input[type="radio"]', root).forEach((r) => {
      r.addEventListener("change", () => {
        const chips = $$(".host-chips", root);
        chips.forEach((c) => (c.style.display = r.value === "hosts" ? "flex" : "none"));
      });
    });
    $$(".host-chip", root).forEach((c) => {
      c.addEventListener("click", () => c.classList.toggle("on"));
    });
  }

  /* ---------------- 初始化 ---------------- */

  function init() {
    renderNav();
    renderEvents();
    renderBans();

    /* 事件筛选 */
    $("#events-realtime").addEventListener("change", (e) =>
      toast(e.target.checked ? "已开启实时刷新（SSE）" : "已关闭实时刷新")
    );
    ["#f-action", "#f-host"].forEach((sel) =>
      $(sel).addEventListener("change", () => {
        state.selectedEvents.clear();
        renderEvents();
      })
    );
    ["#f-ip", "#f-rule"].forEach((sel) =>
      $(sel).addEventListener("input", () => {
        state.selectedEvents.clear();
        renderEvents();
      })
    );
    $("#btn-events-reset").addEventListener("click", () => {
      $("#f-action").value = "all";
      $("#f-host").value = "";
      $("#f-ip").value = "";
      $("#f-rule").value = "";
      state.selectedEvents.clear();
      renderEvents();
    });
    $("#btn-events-refresh").addEventListener("click", () => {
      renderEvents();
      toast("已刷新事件列表");
    });
    $("#events-check-all").addEventListener("change", (e) => {
      const on = e.target.checked;
      $$("#events-body .ev-check").forEach((cb) => {
        cb.checked = on;
        const id = Number(cb.dataset.id);
        if (on) state.selectedEvents.add(id);
        else state.selectedEvents.delete(id);
      });
      syncBulk();
    });
    $("#btn-bulk-ban").addEventListener("click", openBulk);

    /* 封禁名单筛选与 tab */
    $("#ban-search").addEventListener("input", renderBans);
    $("#ban-kind-filter").addEventListener("change", renderBans);
    $$("[data-btab]").forEach((tab) => {
      tab.addEventListener("click", () => {
        state.banTab = tab.dataset.btab;
        $$("[data-btab]").forEach((t) => t.classList.toggle("active", t === tab));
        $("#bans-panel-active").style.display = state.banTab === "active" ? "block" : "none";
        $("#bans-panel-expired").style.display = state.banTab === "expired" ? "block" : "none";
        $("#bans-panel-audit").style.display = state.banTab === "audit" ? "block" : "none";
      });
    });

    /* 打开 / 关闭 */
    $$("[data-open]").forEach((b) => {
      b.addEventListener("click", () => {
        if (b.dataset.open === "ban-create") openBanCreate();
      });
    });
    $$("[data-close]").forEach((b) => {
      b.addEventListener("click", () => {
        const ov = b.closest(".drawer-overlay");
        if (ov) close(ov);
      });
    });
    $$(".drawer-overlay").forEach((ov) => {
      ov.addEventListener("click", (e) => {
        if (e.target === ov) close(ov);
      });
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") $$(".drawer-overlay").forEach(close);
    });

    /* 从事件封禁 */
    ["#fe-ttl"].forEach((id) => bindSegmented($(id)));
    bindScope(drawerFromEvent());
    $("#fe-submit").addEventListener("click", () => {
      const el = $("#fe-submit");
      if (el.disabled) return;
      const kind = el.dataset.kind || "ip";
      const value = el.dataset.target || "";
      const ttl = $("#fe-ttl").dataset.value || "24h";
      addBan(kind, value, ttl, $("#fe-reason").value.trim());
      close(drawerFromEvent());
      navigate("bans");
    });

    /* 新增封禁 */
    bindSegmented($("#bc-ttl"));
    bindScope(drawerBanCreate());
    $$("#bc-kind .seg").forEach((s) => {
      s.addEventListener("click", () => {
        $$("#bc-kind .seg").forEach((x) => x.classList.remove("active"));
        s.classList.add("active");
        createKind = s.dataset.kind;
        $("#bc-value").value = "";
        applyCreateKind();
      });
    });
    $("#bc-value").addEventListener("input", validateCreateValue);
    $("#bc-submit").addEventListener("click", () => {
      if ($("#bc-submit").disabled) return;
      const value = $("#bc-value").value.trim();
      if (!value) {
        $("#bc-value-field").classList.add("invalid");
        return;
      }
      if (createKind === "ip" && !validIPValue(value)) {
        $("#bc-value-field").classList.add("invalid");
        return;
      }
      const ttl = $("#bc-ttl").dataset.value || "24h";
      addBan(createKind, value, ttl, $("#bc-reason").value.trim() || "手动封禁");
      close(drawerBanCreate());
      navigate("bans");
    });

    /* 批量 */
    bindSegmented($("#bulk-ttl"));
    $("#bulk-submit").addEventListener("click", () => {
      const ids = Array.from(state.selectedEvents);
      const ips = [];
      ids.forEach((id) => {
        const e = MOCK.events.find((x) => x.id === id);
        if (e && ips.indexOf(e.ip) < 0) ips.push(e.ip);
      });
      const skipped = ips.filter(isTrusted);
      const ttl = $("#bulk-ttl").dataset.value || "24h";
      const reason = $("#bulk-reason").value.trim() || "批量封禁";
      ips.filter((ip) => !isTrusted(ip)).forEach((ip) => addBan("ip", ip, ttl, reason, true));
      state.selectedEvents.clear();
      close($("#drawer-bulk"));
      renderEvents();
      renderBans();
      navigate("bans");
      toast(
        "已封禁 " + (ips.length - skipped.length) + " 个 IP" +
          (skipped.length ? "，跳过 " + skipped.length + " 个受限地址" : "")
      );
    });

    /* 封禁详情操作 */
    $("#bd-unban").addEventListener("click", () => {
      if (state.activeBanId != null) removeBan(state.activeBanId);
    });
    $("#bd-renew").addEventListener("click", () => {
      if (state.activeBanId != null) renewBan(state.activeBanId);
    });

    /* 深链 */
    const hash = (location.hash || "#events").slice(1);
    navigate(hash === "bans" ? "bans" : "events");
    window.addEventListener("hashchange", () => {
      const h = (location.hash || "#events").slice(1);
      if (h === "events" || h === "bans") navigate(h);
    });
  }

  function addBan(kind, value, ttl, reason, silent) {
    const isCidr = kind === "ip" && value.indexOf("/") >= 0;
    const ttlMap = { "1h": "剩余 1h", "24h": "剩余 24h", "7d": "剩余 7d", forever: "永久" };
    const expMap = { "1h": "12:30", "24h": "11-08 11:30", "7d": "11-14 11:30", forever: "永久" };
    const entry = {
      id: Math.max.apply(null, state.bans.map((b) => b.id).concat([0])) + 1,
      kind: kind === "ua" ? "ua" : isCidr ? "cidr" : "ip",
      label: kind === "ua" ? "UA" : isCidr ? "CIDR" : "IP",
      value: value,
      scope: "全局",
      scopeHosts: [],
      expiresAt: expMap[ttl] || "11-08 11:30",
      ttlLabel: ttlMap[ttl] || "剩余 24h",
      hits: 0,
      actor: "admin",
      reason: reason,
      createdAt: "2026-11-07 " + nowHHMM() + ":00",
      source: "manual",
      ua: kind === "ua" ? $("#bc-ua-mode") ? $("#bc-ua-mode").value : "contains" : "—",
      lastHit: "—",
    };
    state.bans.unshift(entry);
    state.audit.unshift({
      time: nowHHMM(),
      action: "waf.ban.add",
      detail:
        "新增封禁 " + (kind === "ua" ? "ua" : "ip") + "=" + value + " ttl=" + ttl + " scope=global reason=" + reason,
      actor: "admin",
    });
    renderBans();
    if (!silent) toast("已封禁 " + value + "，立即生效");
  }

  document.addEventListener("DOMContentLoaded", init);
})();
