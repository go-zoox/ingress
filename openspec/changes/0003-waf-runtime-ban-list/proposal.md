# change: 0003-waf-runtime-ban-list（WAF 运行时封禁名单：IP + UA）

## 为什么

后台目前**只能看 WAF 拦截记录，不能处置**。运维看到一个 IP 正在暴力扫描，唯一能做的动作是：

1. 手工编辑 `ingress.yaml` 加 `waf.deny`，走「配置 → 保存 → 发布」；
2. 或者干脆什么都不做，等攻击自己停。

这条路有三个硬伤：

- **代价与收益严重不匹配**：现有配置链路是「前端合并整份 YAML → 校验 → `os.WriteFile` → 全量 `prepare()`」。`prepare()` 会重编译 router/connectProxy/backendCache/waf/ratelimit/security/errorPages/maintenance 共十余项，并**清空整个共享缓存**（`core/prepare.go:166` 的 `app.Cache().Clear()`）与**重置所有限流计数器**（`core/prepare.go:78` 重建限流器）。为了封一个 IP 让全站缓存抖动、限流窗口归零，这个取舍不成立。
- **封禁是「配置态」而非「运维态」**：封禁天然是临时的、有生命周期的、需要审计的处置动作，把它写进版本化的 YAML 里，既污染配置 diff，又难以表达 TTL。
- **没有 UA 维度**：`waf.deny` 只接受 IP/CIDR（`core/waf/compile.go` 的 `parseIPCIDR` 解析失败直接报错）。UA 封禁在签名规则里**能力上早已可行**（`header:User-Agent` + `contains`/`regex`，内置 `builtin:scanner-ua` 就是范例），但没有任何「名单」形态，运维得手写一条 WAF 规则并理解 `targets` 语法。

同时，`waf_event` 表里已经有现成的 `client_ip` 与 `user_agent` 字段——**「看到攻击者」和「处置攻击者」之间缺的只是一条通路**。

产品原型 `prototypes/waf-ban/` 已验证交互方向。

## 做什么

### 1. 运行时封禁名单（backend）

- 新增 SQLite 表 `waf_ban` 作为唯一事实源；
- 新增 `core/waf.BanSet` 编译产物 + `CompileBans`，与现有 `Profile` 完全独立；
- `core` 持有 `atomic.Pointer[BanSet]`，读写无锁；
- 封禁条目变更后**只原子替换指针**，不触发 `Reload`、不清缓存、不动限流计数器。

### 2. 请求路径上的封禁判定

- 在 `core/build.go` 路由匹配之后、**限流与 WAF 之前**独立判定一次（封禁应返回 403 而非 429，也不应被 `allow` 放行）
- **独立于 `waf.enabled`**：WAF 总开关（含运行时 override）关闭时封禁仍然生效——封禁是兜底手段，不能被「应急关 WAF」连带关掉；
- **ban 优先于 allow 白名单**：承接现有 `IPAllowlistFormFields` 的「临时封禁：即使命中 allow 也会被拒绝」语义；
- 客户端 IP 复用 WAF 的 `clientIP`（`peerip` 真实 TCP peer + `trusted_proxies`），**不复用 ratelimit 那套**（它读 `RemoteAddr`/XFF，同一请求会算出不同 IP）。

### 3. 封禁类型

| kind | 说明 | 匹配方式 |
|---|---|---|
| `ip` | 单个 IP | 精确匹配（归一化 `To4`，`::ffff:1.2.3.4` → `1.2.3.4`） |
| `ip_cidr` | CIDR 网段 | `net.IPNet.Contains` |
| `ua_contains` | UA 子串 | `strings.Contains`（大小写不敏感） |
| `ua_regex` | UA 正则 | 预编译 `regexp` |

- 每条可带 `hosts[]`：空 = 全局生效，非空 = 仅这些 host（沿用 `CompileHostPatternList` 的 exact/wildcard/regex 推理）；
- 每条带 `expires_at`（nil = 永久）；判定期惰性过滤过期条目。

### 4. Admin API

```
GET    /api/v1/waf/bans              列表（active / expired / all）
POST   /api/v1/waf/bans              新增
PATCH  /api/v1/waf/bans/:id          改 TTL / 启停
DELETE /api/v1/waf/bans/:id          解封
POST   /api/v1/waf/bans/preview      封禁前影响面试算
```

- 写操作全部落审计：`waf.ban.add` / `waf.ban.remove` / `waf.ban.renew` / `waf.ban.expire`，`actor` 取真实登录用户（`auth.UsernameFromContext`），不再硬编码 `"admin"`。

### 5. Admin Web

- WAF 页事件表每行新增「封禁 IP」「封禁 UA」；
- 新增「封禁名单」页面：生效中 / 已过期 / 操作审计；
- 4 个抽屉（从事件封禁 / 新增封禁 / 详情 / 批量），复用现有 `.drawer` 与表单组件；
- 封禁确认前展示影响面（近 24h 命中、涉及 Host 数、独立 UA 数、独立路径数）。

### 6. 危险目标拦截（安全护栏）

- 目标 IP 命中 `waf.trusted_proxies` → **拒绝封禁**（封它等于封掉 LB / 全站），后端返回 400 并在前端禁用提交；
- 同一 IP 近 24h 出现大量不同 UA（默认阈值 20）→ 响应带 `warnings`，前端提示可能误伤 NAT 后的真实用户；
- 永久封禁（`expires_at` 为空）需要显式 `confirm_permanent: true`，否则 400。

### 7. 可观测性

- 访问日志新增 `ban_block=1`（与既有 `waf_block=1` 并列，两个字段独立）；
- WAF 事件表写入 `action=ban`、`rule=ban:ip` / `ban:ua_contains` 等，复用现有事件列表与筛选链路；
- 命中计数：请求路径不写 DB，经 `AsyncRollupRecorder` 式的异步聚合批量回写 `hits`。

## 不做什么

- **不写回 `ingress.yaml`**：封禁不进配置版本体系，靠审计日志追溯（用户定版）；
- **不做自动封禁**：P2 再说。自动封禁是唯一可能造成大面积业务中断的功能，首版不做；
- **不做多副本同步**：Admin 只向本进程 `CoreInstance` 注入。首版明确「单副本有效」，多副本需 Redis pub/sub 或 DB 轮询，属后续；
- **不做 body 扫描**：沿用 WAF v1 无 body 扫描的边界；
- **不改现有 `waf.deny` 语义**：配置态黑名单原样保留，运行时名单是叠加的另一层；
- **不新增 `waf.enabled` 的覆盖逻辑**：`isWAFProfileActive` 的语义不动。

## 影响面

| 层 | 文件 | 变化 |
|---|---|---|
| core waf | 新增 `core/waf/ban.go`、`ban_test.go` | `BanEntry` / `BanSet` / `CompileBans` / `Check` |
| core waf | `core/waf/eval.go` | `CheckRequest` 增加 `opts ...Option`；新增 ban 判定阶段 |
| core | `core/core.go` | `atomic.Pointer[BanSet]` + `SetBanSet` / `GetBanSet` |
| core | `core/build.go` | 路由匹配后独立 ban 检查 + 403 响应 + 日志 |
| core | `core/accesslog.go` | `accessLogMeta` 增 `BanBlock`，输出 `ban_block=` |
| core | `core/waf_callback.go` | `OnWAFEvent` 支持 `action=ban` |
| admin model | `core/admin/model/model.go` | 新增 `WAFBan`，加入 `MigrateModels` |
| admin service | 新增 `core/admin/service/waf_ban.go`、`waf_ban_test.go` | CRUD + 编译推送 + 命中回写 + 审计 |
| admin handler | 新增 `core/admin/handler/waf_ban.go` | 5 个端点 |
| admin bootstrap | `core/admin/app/app.go` | 启动时把 DB 里的 active 封禁编译注入 core |
| admin web | 新增 `pages/WAFBansPage.tsx`、`components/ban/*`、`lib/wafBan.ts` | 名单页 + 抽屉 + 校验 |
| admin web | `pages/WAFPage.tsx`、`api/client.ts`、`App.tsx`、`rbacMenuCatalog.ts` | 事件行入口、路由、菜单 |
| admin rbac | `core/admin/service/rbac/menu.go` | 新增 `waf-bans` 菜单（`waf:read` 权限） |
| docs | `docs/guide/waf.md`、`docs/zh/guide/waf.md`、`AGENTS.md` | 使用说明 |
| examples | `examples/waf-ban/` | 可运行示例 |
| tests | `core/waf/ban_test.go`、`core/ban_build_test.go`、`core/admin/service/waf_ban_test.go`、`core/admin/web/src/lib/wafBan.test.ts` | 单测 + 集成 |

## 验收

1. **默认零影响**：不配置任何封禁时，所有既有行为不变（`core/waf` 现有测试全绿，`CheckRequest` 的 49 处测试调用点无需改动）。
2. 封禁 `203.0.113.42` 后：该 IP 请求返回 403，访问日志 `ban_block=1`，WAF 事件表出现 `action=ban rule=ban:ip`。
3. **WAF 总开关关闭时封禁仍生效**：设置 `SetWAFOverride(false)` 后，被封 IP 依然 403。
4. ban 优先于 allow：某 IP 同时命中 `rules[].waf.allow` 与运行时 ban，返回 403。
5. UA 封禁：`ua_contains: sqlmap` 命中 `sqlmap/1.7.2`；`ua_regex: (?i)nikto` 命中 `Nikto/2.5.0`。
6. host 限定：封禁带 `hosts: [api.example.com]` 时，`www.example.com` 不受影响。
7. TTL：`expires_at` 过期后自动不再拦截，列表归入「已过期」。
8. 解封后立即恢复 200，无需 reload；期间共享缓存未被清空，限流计数器未重置。
9. 安全护栏：封禁 `trusted_proxies` 内的 IP 返回 400；永久封禁缺 `confirm_permanent` 返回 400。
10. 审计：每次增删改都有 `waf.ban.*` 记录，`actor` 为真实用户名。
11. `go test ./core/...` 全绿；`cd core/admin/web && pnpm test` 全绿；`pnpm build`（tsc -b + vite）通过。

## 风险与回滚

| 风险 | 说明 | 缓解 |
|---|---|---|
| 误封 NAT 后的真实用户 | 一个出口 IP 后可能上千用户 | 前端影响面提示 + 默认 24h TTL + 优先建议按 UA 封 |
| 封到 LB 自己 | 未配 `trusted_proxies` 时 clientIP 是 LB | 后端硬拒绝 + 前端禁用提交 |
| 多副本不一致 | 只有收到 API 调用的那个副本生效 | 文档明确标注，P2 再做同步 |
| 性能 | 每请求多一次 ban 检查 | IP 精确走 map（O(1)）；CIDR 与 UA 数量级远低于现有 10 条正则；补 benchmark |

**回滚**：删除 `waf_ban` 表数据（或全部 `enabled=false`）即可完全恢复；代码层 `SetBanSet(nil)` 等价于关闭。无配置变更，无 YAML 污染。
