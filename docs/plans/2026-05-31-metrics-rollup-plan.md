# Admin 实时指标 Rollup 实现计划

> **Goal:** 总览指标不再依赖反复 tail 解析 access log；日志仍 SSE 实时 tail，指标走内存 rollup（后续可选 SQLite）。

**Architecture:** ingress 请求完成时写入结构化事件 → `MetricsRollup` 内存环形缓冲 + 分钟索引；Admin `Metrics.Overview` 优先读 rollup，冷启动/24h 回退 tail。日志页继续 `LogStreamer` 按 offset 推新行。

**Tech Stack:** Go (`core` + `core/admin/service`)，现有 `AccessEntry` / `aggregateOverview`，SQLite（Phase 4）。

---

## Phase 1 — 内存 Rollup 存储（当前）

- [x] 计划文档
- [x] `MetricsRollup`：`Record(AccessEntry)`、保留 2h / 15 万条、按窗口导出
- [x] `Metrics.Overview`：`rollup_live` 优先，tail 回退
- [x] 单元测试：窗口过滤、trim、已关闭分钟桶只增不减（同进程内）

**Files:** `core/admin/service/metrics_rollup.go`, `metrics_rollup_test.go`, `metrics.go`

---

## Phase 2 — Core 请求路径回调

- [x] `core/access_metrics.go`：`AccessMetricsEvent` + `AccessMetricsCallback` + `SetAccessMetricsCallback`
- [x] `core/accesslog_emit.go`：`logAccess()`，打 log 同时 `emitAccessMetrics`
- [x] `core/build.go` / `build_security.go` 全部改用 `logAccess`
- [x] Admin `accessMetricsAdapter` 注册

## Phase 3 — 增量日志回填（冷启动 / 无 hook）

- [x] Admin 启动：`BootstrapRollupFromTail()`（rollup 为空时 tail 1h 种子数据）
- [x] `LogStreamer.SetAccessLineHandler`（**仅** `CoreInstance == nil` 时解析新行，防双计）
- [x] 已有 `SetOnAccessLine` → `OverviewStreamer.PushAll()` 保留

---

## Phase 4 — SQLite 分钟桶持久化（可选）

- [x] 表 `metrics_minute_bucket`（minute, s2, s3, s4, s5, waf, cache_hits, …）
- [x] 每分钟 flush；启动加载 26h
- [x] Job `purge_metrics_buckets` 保留策略

**Files:** `core/admin/model/`, migration, `metrics_rollup_persist.go`, `jobs/registry.go`

---

## Phase 5 — 文档与 AGENTS.md

- [x] `docs/guide/admin.md`：指标来源 `rollup_live` vs `access_log`
- [x] `AGENTS.md` 补充 overview metrics 数据路径

---

## 验证

```bash
go test ./core/admin/service/... -run Rollup
go test ./core/... -run AccessMetrics   # Phase 2+
# 本地：ingress + live_traffic.py，总览 5m/15m 桶稳定、SSE 刷新无回退
```

## 非目标（YAGNI）

- 全量 access log 入 DB
- 首版不做跨进程 rollup（Admin 独立进程且无 CoreInstance 时仍 tail）
