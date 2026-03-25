# Match 预编译与请求路径性能优化

> **For agentic workers:** 实现时可配合 @superpowers:subagent-driven-development 或 @superpowers:executing-plans，按任务勾选推进。

---

## 实现状态总览（TODO）

| 状态 | 项 |
|------|-----|
| **已完成** | `routerIndex` + `compileRouterIndex`（`core/compile.go`） |
| **已完成** | `prepare()` 末尾编译路由；失败则启动/Reload 失败（`core/prepare.go`） |
| **已完成** | `core.router` 字段（`core/core.go`） |
| **已完成** | `match()` 热路径使用 `matchHostIndex` + `matchPathWithRouter`；`HostMatcher.ruleIndex` |
| **已完成** | 缓存键 `match.host:v2:`（避免旧缓存结构错位） |
| **已完成** | 公共 API：`MatchHost` / `MatchPath`（`MatchHost` 内部每次调用仍会 compile，仅测试/外部一次性调用；生产请求走 `c.router`） |
| **已完成** | 基准测试 `core/match_bench_test.go` |
| **已完成** | 文档：`docs/guide/routing.md`、`docs/zh/guide/routing.md`（预编译与失败时机）；`docs/guide/caching.md`、`docs/zh/guide/caching.md`（键名 v2、Reload 清缓存） |
| **已完成** | `reload.go`：已用注释说明 `prepare()` → `prepareCache()` 会清空缓存（不再保留误导性的 `@TODO clear cache`） |
| **未实现（可选）** | 进一步减少分配：仅在有 bench 收益时优化 cache key 拼接 |
| **未实现（可选）** | CHANGELOG 中单独列出「非法正则改为启动/重载失败」的运维行为变化 |

**Goal:** 在配置加载/重载阶段完成主机与路径匹配的预编译与索引，使热路径 `match` 使用已编译的 `*regexp.Regexp`，避免每次请求对 pattern 字符串重复编译。

**Architecture（实际实现）:** `routerIndex` 为按配置顺序的 `entries []compiledRuleEntry`（每条规则一项：`exact` / `regex` / `wildcard`），**保留规则先后顺序**；`pathsByRule [][]compiledPath` 存每条规则下各 path 的预编译正则。`prepare()` 在 `New` 与 `Reload` 时构建快照；`match()` 读 `c.router` 与当前 `c.cfg.Rules`（通过 `ruleIndex`）。非法正则在 `compileRouterIndex` 阶段返回错误。

**Tech Stack:** Go 标准库 `regexp`，`core/rule`、`core/service`、`github.com/go-zoox/proxy/utils/rewriter`。

---

## 文件与职责

| 文件 | 职责 |
|------|------|
| `core/core.go` | `router *routerIndex` |
| `core/compile.go` | `compileRouterIndex`、`routerIndex` 类型 |
| `core/match.go` | `matchHostIndex`、`matchPathWithRouter`、`hostMatcherFromMatchedRule`、`pathMatchResult`、`match()`、`MatchHost`、`MatchPath` |
| `core/prepare.go` | 调用 `compileRouterIndex` |
| `core/match_test.go` | 行为测试 |
| `core/match_bench_test.go` | 基准测试 |
| `docs/guide/routing.md` / `docs/zh/guide/routing.md` | 预编译说明、非法正则失败时机 |
| `docs/guide/caching.md` / `docs/zh/guide/caching.md` | `match.host:v2:`、Reload 与缓存清空 |

---

### Task 1: 定义路由快照类型与编译入口

**Files:**
- Create: `core/compile.go`
- Modify: `core/core.go`

- [x] **Step 1: 设计 `routerIndex`**

  实现为按规则顺序的 `entries`（非单独 exact map），与 `MatchHost` 原语义一致。

- [x] **Step 2: 实现 `compileRouterIndex`**

- [x] **Step 3: 在 `core` 中增加字段并在 `prepare()` 调用编译**

- [x] **Step 4: 运行测试** — `go test ./core/...`

- [ ] **Step 5: Commit**（由维护者按需执行）

---

### Task 2: 用快照重写 MatchHost / MatchPath 并接入 match()

**Files:**
- Modify: `core/match.go`

- [x] **Step 1: 与现有 `match_test.go` 行为对齐**

- [x] **Step 2–3: `matchHostIndex` + `matchPathWithRouter`**

- [x] **Step 4: `match()` 使用 `c.router`；缓存键 `match.host:v2:`**

- [x] **Step 5: 运行测试** — `go test ./...`

- [ ] **Step 6: Commit**（由维护者按需执行）

---

### Task 3: 基准测试与可选微优化

**Files:**
- `core/match_bench_test.go`

- [x] **Step 1: 添加 Benchmark**（`BenchmarkMatchHostIndex_lastExactRule`、`BenchmarkMatchPathWithRouter_lastPath`）

- [ ] **Step 2（可选）: Cache key 进一步减分配**

- [ ] **Step 3: Commit**（由维护者按需执行）

---

### Task 4: Reload 与缓存一致性（可选）

**Files:**
- `core/reload.go`

- [x] **Step 1: 评估** — `prepare()` 内 `prepareCache()` 已对缓存 `Clear()`，Reload 路径已覆盖。

- [x] **Step 2: `reload.go` 注释** — 说明清缓存由 `prepare()` 完成

- [ ] **Step 3: Commit**（可选）

---

### Task 5: 文档

**Files:**
- `docs/guide/routing.md`, `docs/zh/guide/routing.md`
- `docs/guide/caching.md`, `docs/zh/guide/caching.md`

- [x] 预编译、规则顺序、非法正则启动/重载失败、`match.host:v2` 与 Reload 清缓存说明

- [ ] **CHANGELOG 行为说明**（可选）

---

## 风险与回滚

- **行为变化**：非法正则从「延迟失败」变为「启动/重载失败」—— 已在路由文档说明；CHANGELOG 可选。
- **内存**：预编译正则增加常驻内存；规则极大时需监控。
- **回滚**：保留 git 历史回退。

---

## 验收标准

1. `go test ./...` 通过（`-race` 较慢，CI 或本地按需）。
2. 路由行为与优化前一致（现有测试覆盖）。
3. Benchmark 可运行：`go test ./core -bench=BenchmarkMatch -benchmem -run=^$`

---

**计划文件路径:** `docs/superpowers/plans/2026-03-25-match-precompile-performance.md`
