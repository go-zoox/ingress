# design: 0003-waf-runtime-ban-list

> 结论先行：**封禁是运维态，不是配置态**。它需要一个独立的、可在毫秒级全量替换的编译产物（`BanSet`），通过 `atomic.Pointer` 挂到 core 上，在请求路径上做一次**独立于 WAF 开关**的判定。全程不碰 `Reload`、不清缓存、不动限流计数器。

## 1. 为什么不能复用现有机制

### 1.1 不能塞进 `Profile`

`core/waf/compile.go:36-53` 的 `Profile` 所有匹配字段都是**未导出**的：

```go
type Profile struct {
    Enabled          bool
    ...
    denyNet        []*net.IPNet   // 未导出
    allowNet       []*net.IPNet   // 未导出
    signatureRules []*sigRule     // 未导出
}
```

包外无法构造或修改，只能经 `CompileIngress` → `compileProfile`（包内私有）产生。而且 `Profile` 在 `prepare()` 时编译一次，**生命周期与配置绑定**——想动态改就得重新编译整个 `CompileIngress`，语义上就滑向 reload 了。

### 1.2 不能走 `Reload`

`core/reload.go:8-20` 的 `Reload` → `prepare()` 会重编译 12 项并副作用：

| 副作用 | 位置 | 后果 |
|---|---|---|
| 清空共享缓存 | `core/prepare.go:166` `app.Cache().Clear()` | 响应缓存 + match host 缓存全部失效 |
| 重置限流计数器 | `core/prepare.go:78-87` `ratelimit.Compile` | 所有限流窗口归零，攻击者借机放大 |
| 关闭代理空闲连接 | `core/prepare.go:65` | CONNECT 连接池抖动 |
| `c.cfg` 无锁赋值 | `core/reload.go:9` | 与请求路径存在既有竞争窗口 |

封一次 IP 触发上述全部副作用，不可接受。

### 1.3 不能照抄 `SetWAFOverride` 的锁模式

`core/core.go:106-110` 的 `SetWAFOverride` 用 `sync.RWMutex`，且 `isWAFProfileActive`（`core/core.go:143-145`）在 override 非 nil 时**完全绕开 profile**。如果 ban 挂在这条路径上，一旦运维把 WAF 应急关掉，封禁也失效了——这是产品上不可接受的（封禁是兜底）。

另外 `SetWAFCallback`（`core/core.go:149-152`）是**裸赋值**，请求路径无锁读，本身存在 data race 隐患。新代码不复制这个错误。

## 2. 数据流

```text
┌─ 写侧 ──────────────────────────────────────────────┐
│ POST/PATCH/DELETE /api/v1/waf/bans                  │
│   ↓ 校验（IP 归一化 / trusted_proxies 拦截 / TTL）   │
│   ↓ 写 SQLite waf_ban（唯一事实源）                  │
│   ↓ BanService.reloadSnapshot()                     │
│       SELECT * FROM waf_ban WHERE enabled=1         │
│       waf.CompileBans(rows) → (*BanSet, error)      │
│   ↓ CoreInstance.SetBanSet(snap)                    │
│       atomic.Pointer.Store —— 无锁                   │
└─────────────────────────────────────────────────────┘

┌─ 读侧（每请求） ────────────────────────────────────┐
│ build.go: match() 成功                               │
│   ↓ banSet := c.GetBanSet()  ← atomic.Load，无锁      │
│   ↓ waf.CheckBans(banSet, r, hostname, path, ...)   │
│       命中 → 403 + ban_block=1 + WAFCallback(ban)    │
│       未命中 → 继续 rate_limit → WAF → …             │
└─────────────────────────────────────────────────────┘
```

## 3. `BanSet` 设计

```go
// core/waf/ban.go

type BanKind string

const (
    BanKindIP         BanKind = "ip"
    BanKindIPCIDR     BanKind = "ip_cidr"
    BanKindUAContains BanKind = "ua_contains"
    BanKindUARegex    BanKind = "ua_regex"
)

// BanEntry 是编译输入（对应 DB 行）。
type BanEntry struct {
    ID        uint
    Kind      BanKind
    Value     string
    Hosts     []string
    ExpiresAt time.Time // 零值 = 永久
}

// BanSet 是不可变编译产物，整体替换。
type BanSet struct {
    ipExact map[string][]compiledBan // O(1)
    ipNets  []compiledBan            // 线性
    uaRules []compiledBan            // 线性
}

type compiledBan struct {
    id        uint
    label     string          // "ban:ip" / "ban:ua_contains"…，用于事件
    hosts     HostPatternList // 空 = 全局
    expiresAt time.Time
    // 以下按 kind 二选一
    uaMatch   func(string) bool  // contains / regex
}
```

### 3.1 时间语义与时区陷阱

**不要把 `expiresAt` 编进 `BanSet` 做过滤**——那样过期瞬间需要重新编译。改为：

- `CompileBans` 编译**所有 `enabled=1` 的行**，不筛 TTL；
- `Check` 时比较 `time.Now().After(entry.expiresAt)`，零值视为永久；
- 过期条目由每日 job 物理清理 + 列表页按 SQL 过滤。

这样过期是「自动生效」的，不需要任何后台任务参与才能止血/解封。

### 3.2 IP 归一化

`net.ParseIP("::ffff:1.2.3.4")` 与 `net.ParseIP("1.2.3.4")` 是不同表示但同一地址。编译与判定两侧都统一 `if v4 := ip.To4(); v4 != nil { 用 v4 }`，保证 `ip` 类型用 `map[string]` 精确查找是对的。

### 3.3 判定顺序

```text
Check(set, r, clientIP, hostname) (hit *BanEntry, ok bool):
    for each 条目（任意顺序）:
        if 条目.hosts 非空 且 !条目.hosts.Matches(hostname): continue
        if 条目已过期: continue
        switch 条目.kind:
            ip / ip_cidr → 比较 clientIP
            ua_*         → 取 r.UserAgent() 后匹配
        命中 → return 条目
    return nil, false
```

**host 匹配用现有 `HostPatternList`**（`core/waf/host_match.go:124`），它的 exact/wildcard/regex 推理与路由完全一致，避免出现「封了 `api.example.com` 但写成 `*.example.com` 就不匹配」的意外。

### 3.4 性能

| 类型 | 复杂度 | 说明 |
|---|---|---|
| `ip` | O(1) | map 查找 |
| `ip_cidr` | O(n) | `IPNet.Contains`，每次 4 字节 mask-and-compare |
| `ua_*` | O(n) | 预编译 regex / `strings.Contains` |

调研结论：当前一次 `CheckRequest` 已是 10~15 次正则匹配；几千条 `IPNet.Contains` 的成本与现有 signature 检查同量级或更低。**补 benchmark 兜底**（`BenchmarkBanSetCheck`，1000 条 IP + 100 条 CIDR + 50 条 UA）。

## 4. 与 `CheckRequest` 的接入方式

`core/waf/eval.go:14` 的签名有 **49 处测试调用点**（分布在 7 个测试文件）。改成必填位置参数要改 49 处；改成**可变参数选项**则零改动：

```go
type CheckOption func(*checkConfig)

func WithBanSet(b *BanSet) CheckOption { ... }

// 保持向后兼容：不传 Option 时行为与今天完全一致
func CheckRequest(p *Profile, r *http.Request, hostname, path, method string,
    reportFn func(action, rule, cliIP string), opts ...CheckOption) bool
```

这样 `go vet` 不会报错，49 处测试一行不改，且新能力是显式 opt-in。

### 4.1 ban 判定的位置

放在 `CheckRequest` **最前面**（`p == nil || !p.Enabled` 之后、`allowHosts` 之前）？不行——`CheckRequest` 在 `p.Enabled=false` 时直接返回 false，而 ban 必须独立于 WAF 开关。

**正确做法**：`core/build.go` 里**单独调用** `waf.CheckBans`，不塞进 `CheckRequest`：

```go
// build.go，match() 成功后、ratelimit 之前
if c.banSet != nil {
    if entry, ok := waf.CheckBans(c.banSet.Load(), ctx.Request, hostname, path); ok {
        // 403 + ban_block=1 + callback(action=ban)
        return false, true, nil
    }
}
```

理由：
1. ban 与 WAF 生命周期完全解耦（WAF 关了 ban 还在）；
2. 顺序明确（ban 在限流之前，返回 403 而不是 429）；
3. `CheckRequest` 只需加可选的 ban 参数用于**试匹配工具**（`waf_trial.go`）复用，不影响实际拦截。

因此 `CheckRequest` 的 `opts ...CheckOption` 实际上服务于「admin 试匹配时是否把 ban 算进去」。若实现后发现没有消费方，则**不加这个参数**，保持 `CheckRequest` 签名不动——**优先最小改动**。

## 5. core 集成

```go
// core/core.go
type core struct {
    ...
    banSet atomic.Pointer[waf.BanSet]
}

func (c *core) SetBanSet(b *waf.BanSet) { c.banSet.Store(b) }
func (c *core) GetBanSet() *waf.BanSet { return c.banSet.Load() }
```

接口 `Core` 增加这两个方法。**注意**：`atomic.Pointer` 零值可用，不需要初始化。

`core/build.go` 的接缝点（当前 line ~140，`wafProf` 选择之前）：

```go
// 运行时封禁：独立于 WAF 开关与 allow 白名单，优先于限流
if banSet := c.banSet.Load(); banSet != nil {
    if entry := banSet.Check(ctx.Request, hostname); entry != nil {
        // 写 403（复用 error_pages，与 WAF 拦截同形）
        // access log: BanBlock=true
        // callback: OnWAFEvent("ban", entry.Label, host, path, clientIP, ua)
        return false, true, nil
    }
}
```

`clientIP` 从哪来？`waf.BanSet.Check` 内部需要它做 IP 匹配。方案：**`Check` 接收 `*http.Request`，内部调用 `waf.clientIP`**——但 `clientIP` 是包内私有且依赖 `Profile` 的 trust 配置。改为：`BanSet` 编译时**固定使用全局 WAF 的 `trusted_proxies`**（封禁是全局运维动作，不该随路由变化），由 `CompileBans` 接收 `trustedNets []*net.IPNet` 参数。

> 这带来一个启动顺序问题：`prepare()` 失败时 `wafFallback` 为 nil。约束：`CompileBans` 的 `trustedNets` 从 `cfg.WAF.TrustedProxies` 直接解析，不依赖 `Profile`。

## 6. Admin 后端

### 6.1 模型

```go
type WAFBan struct {
    ID        uint       `gorm:"primaryKey"`
    Kind      string     `gorm:"size:16;index"`
    Value     string     `gorm:"size:255;index"`
    Hosts     string     `gorm:"type:text"`     // JSON array
    Reason    string     `gorm:"size:512"`
    Source    string     `gorm:"size:16"`       // manual | auto
    Actor     string     `gorm:"size:128"`
    Hits      int64      `gorm:"default:0"`
    Enabled   bool       `gorm:"index"`
    ExpiresAt *time.Time `gorm:"index"`
    CreatedAt time.Time  `gorm:"index"`
    UpdatedAt time.Time
}
```

`Hosts` 用 JSON 字符串而非关联表：v1 只需读回成 `[]string`，没有按 host 反查的需求。

### 6.2 服务

```go
type WAFBanService struct {
    ingress *Ingress
    audit   *Audit
}

func (s *WAFBanService) SyncToCore() error     // 全量重编译并推到 CoreInstance
func (s *WAFBanService) Create(in BanInput) (*model.WAFBan, error)
func (s *WAFBanService) Update(id uint, in BanPatch) (*model.WAFBan, error)
func (s *WAFBanService) Delete(id uint) error
func (s *WAFBanService) List(filter BanFilter) ([]BanRow, error)
func (s *WAFBanService) Preview(in BanInput) (BanPreview, error)
```

**每次写操作后同步调 `SyncToCore`**（不在事务里异步）：数据量小（几百条），编译是微秒级；同步执行保证「API 返回 200 时封禁已生效」，避免「返回成功但没生效」的诡异体验。

### 6.3 安全护栏（后端强制，不依赖前端）

| 校验 | 行为 |
|---|---|
| IP 命中 `waf.trusted_proxies` | 400，message 前缀 `trusted_proxy:` |
| 永久封禁（`expires_at` nil）且未传 `confirm_permanent` | 400，message 前缀 `confirm_required:` |
| `kind=ua_regex` 但正则编译失败 | 400，message 前缀 `invalid_regex:` |
| `kind=ip` 但 IP 非法 | 400，message 前缀 `invalid_ip:` |
| 已有同 kind+value+hosts 的 active 条目 | 400，message 前缀 `duplicate:` |

> 响应信封沿用现有 `handler/response.go` 的 `{code, message, result}`（`code` = HTTP 状态码），
> 不引入第二套错误码体系；前端靠 `message` 前缀分支。

### 6.4 命中计数

请求路径**不写 DB**（会拖垮热路径）。方案：`core` 侧维护进程内 `sync.Map[uint]int64` 计数器，由 admin 侧每 30s 读走并 `UPDATE ... SET hits = hits + N`，同时清零本地计数。这样即使 DB 抖动也不影响拦截。

### 6.5 启动时注入

`core/admin/app/app.go` 在注册 `SetWAFCallback` 附近调用 `banSvc.SyncToCore()`。若 `CoreInstance == nil`（admin 单独跑），跳过并 log。

### 6.6 审计 actor

现状 `audit.Record("config.save", hash, "admin")` 全是硬编码。新代码用 `auth.UsernameFromContext(ctx)`（`core/admin/auth/auth.go:130`）取真实用户，handler 层传入 service。

## 7. 前端

### 7.1 文件

| 文件 | 说明 |
|---|---|
| `pages/WAFBansPage.tsx` | 名单页：3 tab + 列表 + 抽屉编排 |
| `components/ban/BanDrawer.tsx` | 新增/从事件封禁（同一组件两模式） |
| `components/ban/BanDetailDrawer.tsx` | 详情 + 续期 + 解封 |
| `components/ban/BanBulkDrawer.tsx` | 批量封 IP |
| `lib/wafBan.ts` | 类型、校验（IP/CIDR/regex）、TTL 计算、影响面文案 |
| `lib/wafBan.test.ts` | 纯函数单测 |

### 7.2 事件页改动（最小侵入）

`pages/WAFPage.tsx` 事件表 `<td className="col-actions">` 内追加两个按钮。只改这一处 DOM，不动表格结构与其他列。

### 7.3 视觉

复用 `prototypes/waf-ban/` 已验证的布局与现有 CSS 类（`.panel`、`table.data`、`.drawer`、`.filter-grid`、`.badge`）。新增样式只在必要时补，优先复用。

## 8. 风险与边界

1. **多副本**：只有收到 API 的那个副本生效。文档明确写「单副本有效」，P2 再加 Redis pub/sub。
2. **UA 可绕过**：文档与 UI 都标注为止血手段。
3. **误封 NAT**：默认 24h TTL；预览接口返回 `distinct_ua_24h` 供前端提示。
4. **`ban_block` 与 `waf_block` 的关系**：两个独立字段。ban 命中时 `ban_block=1 waf_block=0`——这样在日志里能区分配置态拦截与运行时封禁。
5. **`waf_event.action`** 新增取值 `ban`：现有前端 `badge-block` / `badge-audit` 需补一个 `badge-ban` 样式；事件筛选下拉需加 `ban` 选项。
6. **既有测试兼容**：`core/accesslog_test.go` 断言了 `waf_block=0`、`maintenance_block=0` 的存在，新增 `ban_block=` 是**追加**字段，不影响 `strings.Contains` 断言；但需确认没有「整行等值」断言。
