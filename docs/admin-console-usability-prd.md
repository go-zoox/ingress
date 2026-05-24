# Admin Console 易用性提升 PRD

## 项目信息

| 字段 | 值 |
|------|------|
| 语言 | 中文 |
| 前端技术栈 | React 19 + TypeScript + Vite（纯手写 CSS，无 UI 组件库，暗色主题） |
| 后端技术栈 | Go + go-zoox/zoox + GORM + SQLite |
| 项目名 | ingress-admin-usability |

## 原始需求复述

为 ingress 控制器的 admin console 增加易用性功能：配置草稿与撤销机制、SSE 实时推送、拓扑图、一键回滚、健康检查面板、证书到期告警、路由详情页。本版本跳过 P0（安全/认证）。

---

## 产品目标

1. **降低误操作风险** — 配置编辑引入草稿状态与撤销机制，回滚操作从手动改为一键确认，避免直接改线上配置
2. **提升实时感知能力** — WAF 事件、日志 tailing、metrics 从轮询切换为 SSE 推送，减少服务端压力的同时让运维获得秒级感知
3. **以路由为中心的可观测性** — 通过拓扑图和路由详情页，将分散在多个页面的信息（配置、指标、日志、WAF、缓存、健康检查）聚合到路由维度，降低认知负荷

---

## 用户故事

1. **作为运维人员**，我在编辑配置时希望有明确的草稿状态和 Ctrl+Z 撤销能力，以免误操作直接改线上配置
2. **作为运维人员**，我希望 WAF 事件和日志能实时推送到页面，而不是手动刷新或等待定时轮询，以便第一时间发现异常
3. **作为运维人员**，我希望通过拓扑图直观看到 host → path → backend 的关系和健康状态，以便快速理解复杂配置
4. **作为运维人员**，我希望在版本历史中一键回滚到指定版本，带确认弹窗，而不需要手动复制 YAML 再发布
5. **作为运维人员**，我点击某条路由后能进入详情页，看到该路由的完整配置、实时指标、日志、WAF 事件、缓存命中率和健康检查状态

---

## 需求池

### P0 — 本版本必须完成

| 编号 | 需求 | 说明 |
|------|------|------|
| P0-1 | 配置草稿 & Undo 机制 | 编辑配置时有 draft 状态标识，支持 Ctrl+Z 撤销，保存/发布前必须显式确认 |
| P0-2 | 证书到期告警 | 总览页证书卡片读取 TLS 真实数据，<30天标黄，<7天标红，移除硬编码 certWarn=0 |
| P0-3 | 路由详情页 | 新增 `/routes/:id` 页面，展示路由完整配置、实时请求指标、日志、WAF 事件、缓存命中率、健康检查状态 |

### P1 — 高价值 UX 提升

| 编号 | 需求 | 说明 |
|------|------|------|
| P1-1 | 实时推送（SSE 替代轮询） | WAF 事件、日志 tailing、metrics 用 SSE 推送，替代 setInterval 轮询 |
| P1-2 | 拓扑/服务关系图 | host → path → backend 拓扑视图，颜色标识健康状态 |
| P1-3 | 一键回滚 | 版本历史中增加"回滚"按钮，带确认弹窗，回滚 = 加载该版本内容 → 校验 → 发布 → reload |

### P2 — 中等价值

| 编号 | 需求 | 说明 |
|------|------|------|
| P2-1 | 健康检查面板 | 定期探测配置了 healthcheck 的 backend，显示 up/down，路由页和拓扑图颜色标识 |
| P2-2 | 总览页版本一致性标识 | 显示当前运行版本 hash vs 最新保存版本 hash，不一致时标黄提醒 |

---

## UI 设计要点

### P0-1：配置草稿 & Undo 机制

**现状**：ConfigPage 已有 `dirty` 标识（`content !== saved`）和 `config-draft-badge`（"草稿未保存"），但无 Undo 栈。

**设计**：
- 在 ConfigPage 中引入 `useUndo` 自定义 hook，维护 `history: string[]` 和 `cursor: number`
- 监听 `onKeyDown`，Ctrl+Z / Cmd+Z 触发 `undo()`，Ctrl+Shift+Z / Cmd+Shift+Z 触发 `redo()`
- YAML textarea 的 `onChange` 将变更推入 history 栈（debounce 300ms 避免每次按键都记录）
- 可视化模块编辑（ConfigModulesPanel）的 `onContentChange` 同样走 undo 栈
- toolbar 区域新增"撤销"/"重做"按钮（图标 + tooltip），与快捷键联动
- 现有 `config-draft-badge` 增强：草稿时显示"草稿未保存（N 处变更）"，保存后显示"已保存"，发布后显示"已发布"

### P0-2：证书到期告警

**现状**：OverviewPage 第 81 行 `const certWarn = 0` 硬编码。

**设计**：
- OverviewPage 加载时同时调用 `api.tlsCerts()` 获取证书列表
- 计算 `certWarn = certs.filter(c => c.days_remaining < 30).length`
- 证书卡片分级显示：
  - 全部 > 30 天：绿色"正常"
  - 存在 < 30 天：黄色"N 需关注"
  - 存在 < 7 天：红色"N 即将过期"
- 证书卡片点击跳转到 `/tls` 页面

### P0-3：路由详情页

**现状**：RoutesPage 展示路由列表和试匹配，无详情入口。

**设计**：
- App.tsx 新增路由 `path="routes/:ruleIndex/:pathIndex"` → `RouteDetailPage`
- RoutesPage 的路由表格每行增加点击事件（整行可点击），跳转到详情页
- 详情页布局采用上下分区：

```
┌─────────────────────────────────────────────────┐
│  PageHeader: 路由详情 — {host}{path}            │
├─────────────────────┬───────────────────────────┤
│  左侧：配置概览      │  右侧：实时指标           │
│  ┌───────────────┐  │  ┌─────────────────────┐  │
│  │ Host: ...     │  │  │ QPS: 1.2k  延迟P50: │  │
│  │ Path: ...     │  │  │ P95: ... 错误率: ... │  │
│  │ Backend: ...  │  │  │ [迷你时间线图]       │  │
│  │ Auth: ...     │  │  └─────────────────────┘  │
│  │ Cache: ...    │  │  ┌─────────────────────┐  │
│  │ HealthCheck:  │  │  │ 健康检查状态         │  │
│  │ WAF: ...      │  │  │ ● UP / ○ DOWN       │  │
│  └───────────────┘  │  └─────────────────────┘  │
├─────────────────────┴───────────────────────────┤
│  Tab 切换：访问日志 | WAF 事件 | 缓存统计        │
│  ┌─────────────────────────────────────────────┐│
│  │ [对应 tab 内容表格/图表]                     ││
│  └─────────────────────────────────────────────┘│
└─────────────────────────────────────────────────┘
```

- **配置概览**：从 `api.routes()` 获取完整路由数据，结合 `api.getConfig()` 解析出 auth/cache/healthcheck/waf 配置
- **实时指标**：新增后端 API `GET /api/v1/routes/:ruleIndex/:pathIndex/metrics`，复用 metrics 聚合逻辑但按 host+path 过滤
- **访问日志 Tab**：复用 `api.logs()` 但自动注入 host 和 path 过滤
- **WAF 事件 Tab**：复用 `api.wafEvents()` 但自动注入 host 和 path 过滤
- **缓存统计 Tab**：复用 `api.cacheOverview()` 的 routes 数据，匹配当前路由展示 ttl/hit_rate
- **健康检查状态**：读取路由配置的 healthcheck 字段，后续 P2-1 接入探测结果

**后端新增 API**：
- `GET /api/v1/routes/:ruleIndex/:pathIndex/metrics` — 按路由聚合指标（QPS、延迟分布、错误率、缓存命中率）
- `GET /api/v1/routes/:ruleIndex/:pathIndex` — 路由完整配置详情（含解析后的 auth/cache/healthcheck/waf）

### P1-1：实时推送（SSE 替代轮询）

**现状**：OverviewPage 用 setInterval 轮询 metrics；WAFPage 用 3s setInterval 轮询事件；LogsPage 用 setInterval 轮询日志。

**设计**：
- 后端新增 SSE endpoint：
  - `GET /api/v1/events/stream` — 统一 SSE 端点，通过 query param 订阅频道：`channels=metrics,waf,logs`
  - 事件类型：`metrics:update`、`waf:event`、`log:line`
- 后端实现：zoox 框架支持 streaming response，设置 `Content-Type: text/event-stream`，`Cache-Control: no-cache`，`Connection: keep-alive`
- 前端新增 `hooks/useSSE.ts`：
  - 封装 `EventSource` 连接管理（自动重连、心跳检测）
  - 返回 `{ data, connected, error }`
  - 页面卸载时自动关闭连接
- 各页面改造：
  - OverviewPage：SSE 替代 `setInterval(fetchMetrics, refreshMs)`
  - WAFPage：SSE 替代 `setInterval(load, 3000)`
  - LogsPage：SSE 替代日志增量轮询
- 降级策略：SSE 连接失败时自动回退到轮询，确保功能不中断

### P1-2：拓扑/服务关系图

**设计**：
- 新增 `/topology` 页面（App.tsx 新增路由，侧边栏新增入口）
- 使用纯 SVG 或 Canvas 绘制拓扑图（不引入第三方图形库，保持零依赖）
- 数据源：`api.routes()` + 配置解析
- 布局：从左到右三层 —— Host 列 → Path 列 → Backend 列
- 节点颜色：
  - 绿色：健康/正常
  - 黄色：有告警（证书即将过期 / 健康检查异常）
  - 红色：故障（健康检查 down / 证书过期）
  - 灰色：未配置健康检查，状态未知
- 交互：点击节点跳转到对应页面（Host → 路由页过滤，Backend → 路由详情页）
- 连线：实线 = 正常，虚线 = redirect/handler，红色闪烁 = 异常

### P1-3：一键回滚

**现状**：ConfigVersionsPanel 的版本详情弹窗有"恢复为草稿"按钮，但只是加载到编辑器，不自动发布。

**设计**：
- 版本详情弹窗 footer 新增"回滚到此版本"按钮（红色/危险色）
- 点击后弹出确认弹窗：
  - 标题："确认回滚到版本 #{id}？"
  - 内容：显示版本 hash、时间、说明，以及 diff 摘要
  - 确认按钮："回滚并发布"（红色）
  - 取消按钮："取消"
- 回滚流程：加载版本内容 → `api.validateConfig()` → `api.publishConfig()` → 刷新页面状态
- 回滚成功后 Toast 提示"已回滚到版本 #{id} 并发布"
- OverviewPage 总览页增加版本一致性卡片（P2-2）

### P2-1：健康检查面板

**设计**：
- 新增后端 API `GET /api/v1/healthcheck` — 返回所有配置了 healthcheck 的 backend 探测结果
- 后端实现：service 层定期（30s间隔）HTTP GET 探测 healthcheck.path，记录 up/down 状态
- 前端新增 `/health` 页面或在现有页面嵌入：
  - 健康检查概览卡片：总数、UP 数、DOWN 数
  - 列表：每行显示 host、path、backend、healthcheck URL、状态（UP/DOWN）、上次探测时间、响应时间
- 路由详情页（P0-3）和拓扑图（P1-2）复用健康检查数据，颜色标识

### P2-2：总览页版本一致性标识

**设计**：
- OverviewPage 的版本卡片增强：
  - 读取 `settings.ingress.config_hash`（运行版本）和 `api.configRevisions()` 最新版本 hash
  - 一致：绿色"配置一致"
  - 不一致：黄色"配置已变更未发布"，附带快捷链接到配置页

---

## 待确认问题

1. **SSE 连接数限制**：单个浏览器可能打开多个 tab，每个 tab 建立一个 SSE 连接。是否需要限制并发连接数？建议后端限制单 IP 最多 5 个 SSE 连接。

2. **Undo 栈深度**：配置 YAML 可能很长，是否限制 undo 栈大小？建议默认 50 步，超过时丢弃最早的记录。

3. **路由详情页的指标数据来源**：当前 metrics 基于访问日志解析（非 Prometheus），按路由过滤需要逐行匹配 host+path，性能是否可接受？如果日志量大（>10000行/15min），是否需要引入内存指标缓存？

4. **健康检查探测频率**：30s 间隔是否合适？后端探测是用 Go goroutine 周期执行还是用 cron 库？探测超时时间建议多少（5s？10s？）

5. **拓扑图布局**：纯 SVG 手写拓扑图在节点数 > 50 时可能布局困难，是否需要引入 dagre/elk 等布局算法？还是限制拓扑图只展示当前页可见范围？

6. **一键回滚的安全边界**：回滚操作是否需要二次输入确认（如输入版本号）？还是弹窗确认即可？回滚后是否自动 reload？

7. **路由详情页 URL 设计**：当前路由数据通过 rule_index + path_index 标识，但这两个值在配置变更后可能变化。是否需要为路由生成稳定的 ID？还是接受 URL 在配置变更后可能失效？
