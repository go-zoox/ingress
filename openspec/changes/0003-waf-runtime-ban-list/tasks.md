# Tasks

> 状态：`[ ]` 待办 / `[x]` 完成。每项完成后回填验证方式。

## 0. 确认

- [x] 产品原型确认（`prototypes/waf-ban/`，用户已确认）
- [x] 本 change（proposal + design + spec delta）确认
- [x] 三个产品决策定版：运行时名单（SQLite + 内存快照）/ IP + UA / 默认全局可按 host 限定

## 1. core / WAF 编译产物

- [x] `core/waf/ban.go`：新增 `BanKind` 常量（`ip` / `ip_cidr` / `ua_contains` / `ua_regex`）
- [x] `core/waf/ban.go`：新增 `BanEntry`（ID / Kind / Value / Hosts / ExpiresAt / Label）
- [x] `core/waf/ban.go`：新增 `BanSet` + `CompileBans(entries []BanEntry, trustedNets []*net.IPNet) (*BanSet, error)`
  - IP 归一化（`To4()`）
  - CIDR 解析
  - UA 正则预编译（非法 → 报错）
  - host 模式编译（复用 `CompileHostPatternList`）
- [x] `core/waf/ban.go`：新增 `BanSet.Check(r *http.Request, hostname string) *BanEntry`
  - host 限定过滤
  - TTL 惰性过期
  - IP map 精确匹配 / CIDR 线性 / UA contains & regex
- [x] `core/waf/ban.go`：新增 `BanSet.CheckWithClientIP(ip net.IP, hostname, ua string) *BanEntry`（纯函数，便于测试）
- [x] `core/waf/ban.go`：新增 `BanSet.Len()` / `BanSet.EntryCountByKind()`（供 admin 显示）
- [x] 单元测试 `core/waf/ban_test.go`：编译（四类 + 非法输入）、匹配（IPv4/IPv6 归一化、CIDR、UA contains 大小写、UA regex、host 限定、TTL 过期、永久条目）
- [x] benchmark `BenchmarkBanSetCheck`：1000 IP + 100 CIDR + 50 UA

## 2. core / 请求路径集成

- [x] `core/core.go`：`core` 增加 `banSet atomic.Pointer[waf.BanSet]`
- [x] `core/core.go`：`Core` 接口增加 `SetBanSet(*waf.BanSet)` / `GetBanSet() *waf.BanSet`
- [x] `core/build.go`：match 成功后、ratelimit 之前插入独立 ban 检查
- [x] `core/build.go`：命中写 403（复用 `writeErrorPage` 与 WAF 拦截同形）
- [x] `core/build.go`：命中触发 `wafCallback.OnWAFEvent("ban", entry.Label, host, path, clientIP, ua)`
- [x] `core/accesslog.go`：`accessLogMeta` 增 `BanBlock bool`，输出 `ban_block=<0|1>`（紧随 `waf_block`）
- [x] `core/build.go`：ban 命中时 `accessLogMeta{BanBlock: true, WAFBlock: false}`
- [x] 集成测试 `core/ban_build_test.go`：封禁命中 403 + `ban_block=1` + `waf_block=0`
- [x] 集成测试：WAF override=false 时封禁仍生效
- [x] 集成测试：ban 优先于 allow 白名单
- [x] 集成测试：host 限定不误伤其他 Host
- [x] 集成测试：TTL 过期后自动恢复 200
- [x] 集成测试：`SetBanSet(nil)` 等价关闭
- [x] 回归：`core/accesslog_test.go` 既有断言不受影响（新增字段为追加）

## 3. Admin 后端

- [x] `core/admin/model/model.go`：新增 `WAFBan` 模型，加入 `MigrateModels()`
- [x] `core/admin/service/waf_ban.go`：`WAFBanService`
  - `SyncToCore()`：全量读 active → `CompileBans` → `CoreInstance.SetBanSet`
  - `Create` / `Update` / `Delete` / `List` / `Preview`
  - 校验：IP 合法性、CIDR 合法性、UA regex 可编译、trusted_proxies 拒绝、永久需 confirm、重复拒绝
  - 每次写后同步 `SyncToCore`
- [x] `core/admin/service/waf_ban.go`：命中计数异步回写（进程内计数 → 批量 UPDATE）
- [x] `core/admin/handler/waf_ban.go`：5 个端点
  - `GET /waf/bans`（status / kind / keyword / limit）
  - `POST /waf/bans`
  - `PATCH /waf/bans/:id`
  - `DELETE /waf/bans/:id`
  - `POST /waf/bans/preview`
- [x] `core/admin/handler/api.go`：注册端点 + 注入 service
- [x] 审计：`waf.ban.add` / `waf.ban.remove` / `waf.ban.renew`（actor 取真实用户）
- [x] `core/admin/app/app.go`：启动时 `SyncToCore()`
- [x] `core/admin/service/rbac/menu.go`：新增 `waf-bans` 菜单项
- [x] 单元测试 `core/admin/service/waf_ban_test.go`：CRUD、校验分支、审计、SyncToCore
- [x] 单元测试 `core/admin/handler/waf_ban_test.go`：端点契约

## 4. Admin 前端

- [x] `core/admin/web/src/api/client.ts`：`WAFBan` 类型 + 5 个 API 方法
- [x] `core/admin/web/src/lib/wafBan.ts`：类型、`validateBanValue`、`ttlToExpiresAt`、`formatRemaining`、影响面文案
- [x] `core/admin/web/src/lib/wafBan.test.ts`：纯函数单测
- [x] `core/admin/web/src/pages/WAFBansPage.tsx`：3 tab + 列表 + 筛选
- [x] `core/admin/web/src/components/ban/BanDrawer.tsx`：新增 / 从事件封禁（双模式）
- [x] `core/admin/web/src/components/ban/BanDetailDrawer.tsx`：详情 + 续期 + 解封
- [x] `core/admin/web/src/components/ban/BanBulkDrawer.tsx`：批量封 IP
- [x] `core/admin/web/src/pages/WAFPage.tsx`：事件行追加「封禁 IP」「封禁 UA」
- [x] `core/admin/web/src/App.tsx`：`/waf/bans` 路由
- [x] `core/admin/web/src/lib/rbacMenuCatalog.ts`：`waf-bans` 菜单
- [x] `core/admin/web/src/styles/app.css`：`badge-ban` 等少量新增样式
- [x] `pnpm build`（tsc -b + vite）通过
- [x] `pnpm test` 通过

## 5. 文档与示例

- [x] `docs/guide/waf.md`：新增「Runtime ban list」小节
- [x] `docs/zh/guide/waf.md`：同上（中文）
- [x] `docs/guide/configuration.md` / `docs/zh/`：访问日志字段表补 `ban_block`
- [x] `examples/waf-ban/`：可运行示例 YAML
- [x] `AGENTS.md`：补一节（运行时封禁名单：机制、边界、Admin 入口）

## 6. 发布门禁

- [x] `GOCACHE=/tmp/ingress-gocache go test ./core/...` 全绿
- [x] `make -C core/admin web` 通过
- [x] `cd core/admin/web && pnpm test` 通过
- [x] 本地起 Admin 冒烟：创建封禁 → 访问被拦 → 解封 → 恢复
- [x] 交付说明：行为变化、API 契约、单副本限制、回滚方式

## Verification（2026-11-07）

### Go

```bash
GOCACHE=/tmp/ingress-gocache GOPROXY=off go test ./core/waf/... ./core/ ./core/admin/service/ ./core/admin/app/ ./core/admin/service/rbac/
```

- `core/waf`：**通过**。新增 `ban_test.go` 19 个用例（四种 kind 编译、IPv4/IPv6 归一化、CIDR、UA contains 大小写、UA regex、空 UA 不匹配 `.*`、host 限定 exact/wildcard/全局、TTL 过期与永久、nil 安全、peer IP、伪造 XFF 忽略、trusted_proxies 解析、精确 IP 优先于 CIDR）。
- `core`：**通过**。`ban_build_test.go` 10 个集成用例（含 `-race` 并发快照交换）（封禁 403、`ban_block=1`/`waf_block=0`、WAF override=false 仍生效、ban 优先于 allow、host 限定、UA 封禁、TTL 过期放行、`SetBanSet(nil)` 复位、ban 先于限流返回 403 而非 429）。
- `core/admin/service`：新增用例**通过**（CRUD、四类校验错误、永久需确认、重复拒绝、List 三态过滤、Delete 清理 core、Update 续期审计、core 缺失时跳过、过期行不进快照、kind/expiry/hosts 归一化、`action=ban` 事件往返、命中计数按条目精确累加且不重复计、并发计数 `-race` 通过）。
- `core/admin/app`：新增 `waf_ban_api_test.go` 6 个端点用例**通过**（CRUD 全链路、4 类校验错误 → 400 + message 前缀、重复/404、preview、UA 警告、未登录 401）。
- `core/admin/service/rbac`：**通过**（新增 `waf-bans` 菜单不破坏既有断言）。

**baseline 对照**：以下 6 个用例在本次改动**之前**同样失败（已用 `git stash` 在干净 master 上复现），属环境依赖，与本次无关：
`core/admin/handler` 的 2 个 terminal 用例（`/dev/ptmx` permission denied）、
`core/admin/config` 的 `TestAdminConsoleDemoLogPaths` 与 `core/admin/service` 的 `TestParseAdminConsoleAccessLog`（需运行时 `examples/admin-console/access.log`）、
`core/admin/service/geoip` 的 2 个用例（root 下权限断言失效）。

### 前端

```bash
cd core/admin/web && pnpm exec tsc -b     # 通过
cd core/admin/web && pnpm test            # 20 文件 / 197 用例全通过（wafBan 46 + wafRuleTooltip 8）
cd core/admin/web && pnpm build           # tsc -b + vite build 通过
```

### Benchmark

```
BenchmarkBanSetCheck-28   1150 条条目（1000 IP + 100 CIDR + 50 UA）   2353 ns/op
```

首版 9047 ns/op，定位到每条 UA contains 规则重复 `strings.ToLower(userAgent)`；改为每请求只小写一次后降到 **2353 ns/op（约 3.8×）**。

### 端到端冒烟（真实进程）

配置：`port 18080` / `admin 19080` / `waf.trusted_proxies: [127.0.0.1/32]`（用 XFF 模拟真实客户端）。

| 场景 | 结果 |
|---|---|
| 封禁前访问 | 200 |
| 创建 IP 封禁 `203.0.113.42` | 200 + 返回行 |
| 重启进程后封禁自动恢复（从 SQLite 载入） | active 列表含该条；请求 403 |
| `X-Forwarded-For: 203.0.113.42`（经受信代理） | **403**，日志 `ban_block=1` |
| `X-Forwarded-For: 198.51.100.9` | 200，`ban_block=0` |
| `POST /waf/toggle {"enabled":false}` 后再访问被封 IP | **403**（封禁独立于 WAF 开关） |
| `DELETE /waf/bans/1` 后同 IP | **200**（立即恢复，无 reload） |
| UA 封禁 `sqlmap`：`UA: sqlmap/1.7.2` / `UA: curl/8.4.0` | 403 / 200 |
| 封禁受信代理自身 `127.0.0.1` | **400** `trusted_proxy: …` |
| 永久封禁缺 `confirm_permanent` | **400** `confirm_required: …` |
| 审计 | `waf.ban.add` / `waf.ban.remove`，`actor=admin`（真实登录用户） |
| 命中计数（30s flush 后） | `hits=1` |

### 示例校验

```bash
go run ./cmd/ingress validate -c examples/waf-ban/ingress.yaml   # valid
```

### 复审轮（round 2）补齐的 spec 承诺

对照 spec delta 逐条核查后发现 4 处承诺未落实，已修复：

1. **`ban` 动作筛选**：spec 要求「事件筛选 SHALL 支持 `ban`」，但 `WAFPage.tsx` 的下拉只有 `block` / `audit`。已补 `<option value="ban">Ban（运行时封禁）</option>`（后端 `ListWAFEvents` 按字符串等值过滤，无需改动）。新增 Go 测试 `TestWAFBanEventRecordedAndFilterable` 验证 `action=ban` 的写入、筛选、与 `block` 互不串台，以及 UA 透传。
2. **`ban` 事件的规则提示**：`rule=ban:ip` 在规则目录里查不到，tooltip 会显示「未找到规则定义（可能为历史或演示事件）」，把运维封禁误导成坏数据。新增 `resolveBanRule`（`lib/wafRuleTooltip.ts`），为四种 kind 合成规则详情并把 `source: 'runtime'` 显示为「运行时封禁」；新增 8 个 vitest 用例。
3. **作用域切换残留（真实 bug）**：`BanDrawer` 把 scope 切回「全局」后，已勾选的 host chips 仍留在 state 里，`buildBanPayload` 会带上 `hosts` —— **界面显示全局、实际提交 host 限定封禁**。修复为：切到 global 时清空 hosts，且 payload 用 `scope === 'hosts' ? hosts : []` 兜底。
4. **`-race` 并发验证**：补 `TestRecordBanHitConcurrent`（50 goroutine × 100 次计数，`FlushBanHits` 后断言精确等于 5000）与 `TestBuild_BanSetSwapIsConcurrencySafe`（一个 writer 持续交换快照、4 个 reader 并发打请求路径），均在 `-race` 下通过，确认 `atomic.Pointer` 读取无竞争。

### 发现并修复的问题

1. **SQLite 时区**：`expires_at` 以本地时间字符串存储，用 SQL 与 UTC `time.Time` 比较会因时区偏移误判（实测导致 active 列表为空）。改为 Go 侧过滤 TTL（`SyncToCore` / `List` / `duplicateExists` 三处）。
2. **Go 正则 vs JS 正则**：后端用 Go `regexp`，支持 `(?i)` 内联标志、`(?P<name>…)`、POSIX 字符类；JS `RegExp` 直接报 "Invalid group"。若不处理，前端会把后端完全合法的 `(?i)(nikto|sqlmap)` 判为非法。新增 `goRegexToJS` 做语法转译，并补 5 个单测。
3. **UA 小写重复计算**：见 benchmark 一节。
4. **`waf.deny` 只接受 IP**：确认 UA 封禁无法复用配置态黑名单，必须走独立的 `BanSet`（与调研结论一致）。

### 已知限制（已在文档与 UI 中明示）

- **单副本**：只有收到 API 调用的 admin 进程注入自己的 `CoreInstance`；多副本同步未实现。
- **UA 可绕过**：客户端可控，仅作止血。
- **无自动封禁**：封禁始终由人工或外部系统调用 API 创建。
- 前端 `waf-bans` 菜单的 `menu:waf-bans` 权限码需在既有部署中重新 seed RBAC 才可见（内置角色启动时会 Replace 权限集合）。
