# Web 应用防火墙（WAF）

Ingress 在 **路由匹配完成之后**、**重定向 / 静态处理 / 回源代理之前**，对请求执行第一代 WAF：

- **IP**：`deny` 先于 `allow`；若配置了非空 `allow`，则仅列表内网段可通过 IP 阶段。
- **域名级 IP 白名单**：`rules[].waf.allow` 按域名限制来源 IP，复用 WAF IP 阶段（见下文「域名级 IP 白名单」）。
- **运行时封禁名单**：在管理后台直接封禁 IP / CIDR / User-Agent，无需改 `ingress.yaml`（见下文「运行时封禁名单」）。
- **域名白名单**：`allow_hosts` 中的 Host 跳过全部 WAF 阶段（IP 与特征均不检查）。
- **单条规则域名白名单**：`rules[].allow_hosts` 中的 Host 仅跳过**该条规则**，其余规则照常评估。
- **特征**：针对 `path`、`query`、`uri`（path + raw query）、`headers`（所有头拼接）或 `header:Name` 做单次匹配；支持 `regex` / `contains`。
- **可选内置规则**：常见 SQL 注入试探、路径穿越、脚本相关试探；用 **`disable_builtin: true`** 关闭。

默认 **不扫请求体**。每条签名规则（含内置）可设 **`action`**：`block`（拦截，默认）、`audit`（只记录）、`pass`（命中后放行并跳过后续签名）。仍可用全局或单条 **`log_only: true`**（等价于未写 `action` 时的 `audit`）。

顶层 **`waf:`** 为类型化基线；**`rules[].waf`** 为 **局部 YAML map**，在 `config.Load` 之后由 **`waf.ApplyRulePatchesFromYAML`**（`core/waf/yaml.go`）按字段合并（解码器无法用同一流程表达“只覆盖部分字段”）。

评估顺序：**合并策略 → enabled → allow_hosts（全局域名白名单）→ 客户端 IP → deny/allow → 特征**（每条特征前检查该规则的 `allow_hosts`）。客户端 IP 判定优先级：**`trusted_proxies`** → **`trust_proxy` + `xff_index`** → TCP 对端。配置 `trusted_proxies` 后，仅当 TCP 对端命中受信网段时才采信 `X-Forwarded-For`，并取**右起第一个非受信地址**；WAF 始终从真实 TCP 对端出发，不会使用被转发头改写过的 `RemoteAddr`。

## 域名级 IP 白名单

用 **`rules[].waf.allow`** 把单个域名限制为内网或指定来源 IP。它复用 WAF 的 IP 阶段，不新增独立模块。

```yaml
rules:
  - host: admin.example.com
    backend:
      service:
        name: admin-console
        port: 8080
    waf:
      enabled: true            # 必须：该路由的 WAF 要启用
      disable_builtin: true    # 仅 IP 准入（不跑攻击签名）
      allow:
        - 10.0.0.0/8
        - 192.168.0.0/16
        - 203.0.113.7/32
      # deny:                  # 可选，优先于 allow
      #   - 203.0.113.66/32
```

语义：

| 配置 | 行为 |
|---|---|
| 没有 `waf`，或 `allow`/`deny` 都为空 | 不限制（**默认**，与现状一致） |
| `allow` 非空 | 仅命中 `allow` 的来源放行，其余返回 **403** |
| `deny` 非空 | 命中即拒绝（即使也在 `allow` 中） |
| `allow` 空 + `deny` 非空 | 纯黑名单（高级用法） |
| 配了 `allow`/`deny` 但该路由 WAF 未启用 | `ingress validate` 直接报错，避免安全配置静默失效 |

客户端 IP 判定（安全关键）：

- **默认**：只认真实 TCP 对端。ingress 会在 zoox `realip` 中间件用 `X-Forwarded-For` / `X-Real-IP` / `True-Client-IP` 改写 `RemoteAddr` **之前**捕获对端地址，客户端无法伪造来源。
- **`waf.trusted_proxies`**（全局）：填写 LB / Ingress Controller 的网段。仅当 TCP 对端命中受信网段时才采信 `X-Forwarded-For`，取**右起第一个非受信地址**。
- **`trust_proxy` + `xff_index`**（旧配置）：保留兼容；新部署建议用 `trusted_proxies`。`trusted_proxies` 里的 `0.0.0.0/0`、`::/0` 会在校验阶段被拒绝。

拦截与可观测性：

- 拦截返回 **403**，复用 [`error_pages`](./configuration.md) 的 403 页面。
- 访问日志：`waf_block=1`。
- WAF 日志：`[waf block] phase=ip allow`（或 `phase=ip deny`），带判定后的 `client_ip`。
- WAF 事件：`rule=ip allow` / `rule=ip deny`（Admin 控制台可筛选）。

Admin 控制台：

- **路由编辑 → IP 白名单**：允许/拒绝列表、继承全局受信代理、403 响应说明，以及**拦截试算**（输入 `RemoteAddr` + `X-Forwarded-For`，给出放行/拒绝与依据）。
- **WAF 模块**：`trusted_proxies` 编辑与「右起第一个非受信地址」算法说明。
- 路由列表与详情展示白名单状态（`allow n` / `未限制`）。

可运行样例：`examples/ip-allowlist/`。

## 内置 starter 规则

未设置 **`disable_builtin: true`** 时，会先加载下表中的规则，再追加你在 `waf.rules` 里自定义的规则。内置 **`id`** 固定，便于在日里识别，也可用 **同名 `id`** 在配置里**覆盖**内置定义（示例：`examples/waf/rule-merge-by-id.yaml`）。正则语义为 [Go `regexp`](https://pkg.go.dev/regexp/syntax)。

| ID | 检测目标（`targets`） | 说明 |
|----|----------------------|------|
| `builtin:sqli-common` | `uri`（path + query） | 常见 SQL 注入试探：`union select`、`sleep(`、`benchmark(`、`; drop/truncate/alter table` 等 |
| `builtin:path-traversal` | `path` | 路径穿越：`../`、`..\`、编码形式的 `..`、`etc/passwd` 等 |
| `builtin:xss-lite` | `uri` | 轻度 XSS 试探：`<script`、`javascript:`、`on…=` 类事件属性 |
| `builtin:rce-probes` | `uri` | 命令注入/RCE 探针 |
| `builtin:jndi-lookup` | `uri`、`headers` | JNDI `${…jndi:` 注入 |
| `builtin:sensitive-files` | `path` | 敏感路径（`.env`、`.git/`、管理后台等） |
| `builtin:ssrf-probes` | `uri` | 云元数据 / `file://` / `gopher://` SSRF 探针（不含 localhost，避免 OAuth 误报） |
| `builtin:scanner-ua` | `header:User-Agent` | 常见扫描器 UA |
| `builtin:crlf-injection` | `uri`、`headers` | CRLF / 响应拆分 |
| `builtin:php-ssti` | `uri` | PHP eval、`php://` 等 |

单条启停：

- **`disable_builtin: false`**（默认）：内置规则默认全开，可在 **`waf.builtin_rules`** 里对某条设为 `false`。
- **`disable_builtin: true`**：内置规则默认全关，可在 **`waf.builtin_rules`** 里单独设为 `true`。
- 自定义 **`waf.rules[]`** 支持 **`enabled: false`**（省略时默认启用）。

示例：

```yaml
waf:
  enabled: true
  builtin_rules:
    builtin:scanner-ua: false
  rules:
    - id: my-rule
      enabled: false
      type: contains
      pattern: /internal
      targets: [path]
```

与代码一致的正则（源码：`core/waf/builtin.go`）：

```
builtin:sqli-common
(?is)(union\s+select\b|sleep\s*\(|benchmark\s*\(|;\s*(drop|truncate|alter)\s+table\b)

builtin:path-traversal
(?:\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\\\|etc/passwd\b)

builtin:xss-lite
(?is)(<\s*script\b|javascript:\s*[a-z]|\bon(?:click|load|error|focus|blur|change|submit|mouse\w*|key\w*|touch\w*|pointer\w*|scroll|dblclick|drag\w*|drop|input|reset|select|wheel|copy|cut|paste|abort|contextmenu|message|unload|beforeunload)\s*=)
```

内置规则可能对正常流量误报，可先用 **`log_only`** 观察，或 **`disable_builtin: true`** 后仅靠自定义规则。常见正常流量回归见 **`core/waf/builtin_false_positive_test.go`**。

## 运行时封禁名单

管理后台可以直接封禁客户端 IP 或 User-Agent，**不需要编辑 `ingress.yaml`**。

这是**运维态**能力，不是配置：封禁条目存在后台数据库（`waf_ban` 表），编译成不可变快照后由运行中的 ingress 原子替换。因此发布一条封禁**不会**：

- 触发配置 reload；
- 清空 HTTP 响应缓存或路由匹配缓存；
- 重置限流计数器；
- 重启监听或断开连接。

### 语义

| 维度 | 行为 |
|---|---|
| 类型 | `ip`（单地址）、`ip_cidr`（网段）、`ua_contains`（子串，大小写不敏感）、`ua_regex`（Go 正则） |
| 范围 | 默认全局；条目可带 `hosts` 只对这些域名生效（exact / wildcard / regex 推理同路由） |
| 生命周期 | 可选 `expires_at`，不填为永久。过期在每次请求时判定，因此到期瞬间即失效，**不依赖任何定时任务** |
| 顺序 | 在**路由匹配之后**、**限流与 WAF 之前**判定。被封客户端拿到 **403**，不会是 **429** |
| WAF 开关 | **独立于 `waf.enabled`**：关闭 WAF（配置或运行时开关）**不会**关闭封禁名单 |
| 白名单 | **优先于 `rules[].waf.allow`**：封禁白名单内的 IP 依然会拦截 |

### 客户端 IP

封禁使用与 WAF **完全相同**的客户端 IP 算法：优先真实 TCP 对端，对端命中 `waf.trusted_proxies` 时才采信 `X-Forwarded-For`。来自非受信来源的伪造 XFF 会被忽略，无法靠改请求头绕过封禁。

### 可观测性

- 访问日志新增 **`ban_block=1`**，与 `waf_block` 是两个独立字段：封禁命中时 `ban_block=1 waf_block=0`，因此可以区分「运维封禁」与「配置态 WAF 拦截」。
- WAF 事件记录 `action=ban`、`rule=ban:<kind>`（如 `ban:ip`、`ban:ua_contains`），事件列表可筛选。
- 每次新增 / 续期 / 解封都会写审计（`waf.ban.add`、`waf.ban.renew`、`waf.ban.remove`），`actor` 为真实操作人。

### 管理后台 API

```text
GET    /api/v1/waf/bans?status=active|expired|all&kind=&keyword=
POST   /api/v1/waf/bans
PATCH  /api/v1/waf/bans/:id
DELETE /api/v1/waf/bans/:id
POST   /api/v1/waf/bans/preview
```

**服务端强制的安全护栏**（前端只是镜像，API 才是权威）：

- 目标 IP 命中 **`waf.trusted_proxies`** 时**拒绝创建**——那是反向代理或负载均衡自身，封掉会导致整站不可访问；
- 创建**永久**封禁必须带 `confirm_permanent: true`；
- `ua_regex` 必须可编译，`ip` / `ip_cidr` 必须可解析；
- 完全相同的生效条目（`kind` + `value` + `hosts`）会被判定为重复并拒绝。

### 限制

- **仅单副本生效**：只有收到 API 调用的那个 admin 进程会把快照注入自己的 ingress 实例。多副本部署下其他副本仍用旧名单；Redis pub/sub 或 DB 轮询同步尚未实现。
- **UA 可绕过**：`User-Agent` 完全由客户端控制，换一个即可绕开。按 UA 封禁只作止血手段，不是防御手段。
- **NAT / 运营商出口**：一个出口 IP 背后可能是上千真实用户。`preview` 接口会返回近 24h 的不同 UA 数量，便于操作前识别风险。
- **不做自动封禁**：封禁始终由人工（或外部系统调用 API）创建。

可运行样例：`examples/waf-ban/`。字段表格见 **[配置参考](./configuration.md)**；英文版说明：**[English WAF](../../guide/waf.md)**。
