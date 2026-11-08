# WAF — spec delta（0003-waf-runtime-ban-list）

> 本文件是本次变更的 **Requirement 增量**。实现完成、验收通过后，把以下 Requirement 合入
> `openspec/specs/waf/spec.md`，并归档 change 目录。

## ADDED Requirements

### Runtime ban list

系统 SHALL 提供独立于配置文件的**运行时封禁名单**，用于应急处置：

- 封禁条目 SHALL 持久化在管理后台数据库（`waf_ban` 表），SHALL NOT 写入 `ingress.yaml`；
- 封禁条目 SHALL 支持四种类型：`ip`（单 IP）、`ip_cidr`（CIDR 网段）、`ua_contains`（UA 子串）、`ua_regex`（UA 正则）；
- 每个条目 SHALL 可携带 `hosts` 列表限定生效域名；为空时 SHALL 对所有 Host 生效；
- `hosts` 条目的模式推理 SHALL 与路由一致（exact / wildcard / regex 自动推断，复用 `HostPatternList`）；
- 每个条目 SHALL 可携带 `expires_at`；为空表示永久；过期后 SHALL 自动停止拦截，无需任何人工或后台任务参与；
- IP 类型条目 SHALL 在编译与判定两侧统一做 IPv4 归一化（`::ffff:1.2.3.4` 与 `1.2.3.4` 视为同一地址）。

### Ban evaluation on the request path

系统 SHALL 在请求路径上执行运行时封禁判定：

- 判定位置 SHALL 在路由匹配成功之后、限流与 WAF 检查之前；
- 命中封禁 SHALL 返回 **403**，并复用 `error_pages` 的 403 页面（与 WAF 拦截同形）；
- 判定 SHALL 独立于 `waf.enabled`：WAF 全局开关关闭或运行时 override 为 false 时，封禁 SHALL 仍然生效；
- 封禁 SHALL 优先于路由级 `rules[].waf.allow` 白名单：同时命中时 SHALL 拒绝；
- 客户端 IP 解析 SHALL 复用 WAF 的算法（真实 TCP peer 优先 + `trusted_proxies` 受信代理），SHALL NOT 使用 `RemoteAddr`/XFF 直读的另一套实现；
- 封禁名单更新 SHALL 通过原子替换编译产物完成，SHALL NOT 触发配置 reload，SHALL NOT 清空响应缓存，SHALL NOT 重置限流计数器。

### Ban list observability

系统 SHALL 复用现有可观测性，不新增独立事件表：

- 访问日志 SHALL 在封禁命中时输出 `ban_block=1`；
- `ban_block` 与 `waf_block` SHALL 是两个独立字段：封禁命中时 `ban_block=1` 且 `waf_block=0`；
- WAF 事件 SHALL 以 `action=ban`、`rule=ban:<kind>`（如 `ban:ip`、`ban:ua_contains`）记录，供 Admin 事件列表筛选；
- 封禁命中 SHALL NOT 写入热路径数据库；命中计数 SHALL 通过进程内计数异步批量回写。

### Ban list admin API

管理后台 SHALL 提供封禁名单的 CRUD API：

- `GET /api/v1/waf/bans`：列表，支持按状态（`active` / `expired` / `all`）、类型、关键字过滤；
- `POST /api/v1/waf/bans`：新增；
- `PATCH /api/v1/waf/bans/:id`：修改 TTL 或启停；
- `DELETE /api/v1/waf/bans/:id`：解封；
- `POST /api/v1/waf/bans/preview`：封禁前影响面试算，SHALL 返回该目标近 24h 命中数、涉及 Host 数、独立 UA 数、独立路径数；
- 写操作返回成功时，封禁 SHALL 已生效（同步编译并注入，非异步）；
- 所有写操作 SHALL 记录审计，动作名为 `waf.ban.add` / `waf.ban.remove` / `waf.ban.renew` / `waf.ban.expire`，`actor` SHALL 为真实登录用户名；
- 响应信封 SHALL 沿用现有 `{code, message, result}` 结构，不引入第二套错误码。

### Ban safety guardrails

系统 SHALL 在后端强制以下安全护栏（不得仅依赖前端）：

- 目标 IP 命中 `waf.trusted_proxies` 时 SHALL 拒绝创建，返回 400 —— 封禁反向代理/负载均衡自身会导致整站不可访问；
- 创建永久封禁（`expires_at` 为空）时 SHALL 要求显式 `confirm_permanent: true`，否则返回 400；
- `ua_regex` 类型的正则 SHALL 在创建时编译校验，失败返回 400；
- `ip` / `ip_cidr` 类型的值 SHALL 在创建时解析校验，失败返回 400；
- 已存在相同 `kind` + `value` + `hosts` 且处于生效中的条目时 SHALL 拒绝重复创建，返回 400。

## MODIFIED Requirements

### WAF event actions

原 `waf_event.action` 取值范围为 `block` | `audit`；本次扩展：

- 新增取值 `ban`，表示请求被运行时封禁名单拦截；
- 既有 `block` / `audit` 语义 SHALL 保持不变；
- 事件列表筛选 SHALL 支持 `ban`。

### Access log fields

原访问日志字段尾部为 `… waf_block=<0|1> rate_limit_block=<0|1> maintenance_block=<0|1> …`；本次扩展：

- 新增 `ban_block=<0|1>` 字段，紧随 `waf_block` 之后输出；
- 既有字段的位置与含义 SHALL 保持不变（新增字段 SHALL NOT 改变既有字段的键名或语义）。

## REMOVED Requirements

无。
