# 定时任务

Ingress 可在 **Admin 控制台** 同一进程内运行 **Cron 定时任务**，用于运维清理（WAF 事件、审计日志）、TLS 巡检、GeoIP 同步，以及轻量 HTTP / 命令集成，无需单独部署调度器。

启用条件：**`admin.enabled: true`** 且 **`admin.database`** 可用（默认 SQLite）。每次执行写入 **`job_run`** 表；界面入口为 **维护 → 定时任务**（`/jobs`）。

## 内置任务与自定义任务

| 来源 | 配置 | Admin UI | 说明 |
|------|------|----------|------|
| **内置**（`source: builtin`） | 可选 `jobs.builtins.<id>` 覆盖 | 可改调度 / 开关 / 参数；不可删除 | 代码注册（`core/admin/service/jobs/registry.go`） |
| **自定义**（`source: config`） | `jobs.items[]` | API / UI 完整 CRUD | `kind`：`http_call` 或 `script`（`command` 为兼容别名） |

### 内置任务

| ID | 默认 cron | 作用 |
|----|-----------|------|
| `purge_waf_events` | `0 3 * * *` | 按 `params.retain_days` 清理过期 WAF 事件（默认 **30** 天） |
| `purge_audit_logs` | `0 4 * * 0` | 按 `params.retain_days` 清理过期审计日志（默认 **90** 天） |
| `check_tls_expiry` | `0 */6 * * *` | 扫描已配置 TLS 证书，过期/即将过期时告警 |
| `sync_geoip` | `0 2 * * *` | 从 `ingress.yaml` 重载 `admin.geoip` |

覆盖示例：

```yaml
jobs:
  builtins:
    purge_waf_events:
      enabled: true
      schedule: "0 3 * * *"
      params:
        retain_days: 14
```

不写 `jobs.builtins` 时使用代码默认值（默认均为启用，可用 `enabled: false` 关闭单项）。

### 自定义任务（`jobs.items[]`）

```yaml
jobs:
  items:
    - id: nightly-health
      name: 夜间健康检查
      kind: http_call
      schedule: "0 1 * * *"
      enabled: true
      timeout_sec: 30
      on_failure: log
      params:
        method: GET
        url: https://backend.internal/healthz
        expect_status: [200]
```

| 字段 | 说明 | 默认 |
|------|------|------|
| `id` | 唯一 ID（必填） | — |
| `name` | 显示名称 | 同 `id` |
| `kind` | `http_call` 或 `script`（`command` 为兼容别名） | — |
| `schedule` | 5 段 cron 表达式 | 必填 |
| `enabled` | 为 `true` 时注册到调度器 | YAML 未写时按解码结果 |
| `timeout_sec` | 单次超时（秒） | **60** |
| `on_failure` | `log` / `retry` / `disable` | `log` |
| `params` | 见下文 | — |
| `params.engine` | `shell`（默认）/ `javascript` / `go` | `shell` |

**`on_failure` 行为**

- **`log`** — 写入 `job_run` 与审计（`job_run`），等待下次 cron。
- **`disable`** — 仅 **自定义** 任务：将 YAML 中 `enabled` 设为 `false` 并重新加载调度。
- **`retry`** — 配置可写；**不会立即重跑**，仍等下次 cron。

## Cron 表达式

使用 zoox 内置 cron（标准 **5 段**：`分 时 日 月 周`）。

| 表达式 | 含义 |
|--------|------|
| `0 3 * * *` | 每天 03:00 |
| `0 4 * * 0` | 每周日 04:00 |
| `*/15 * * * *` | 每 15 分钟 |
| `0 */6 * * *` | 每 6 小时 |

自定义任务保存时空 `schedule` 会报错；内置任务 `schedule` 为空时保留内置默认值。

## `http_call` 任务

```yaml
params:
  method: POST
  url: https://api.example.com/v1/export
  headers:
    Authorization: Bearer ${TOKEN}
  body: '{"window":"daily"}'
  expect_status: [200, 202]
  insecure_tls: false
```

| 参数 | 说明 |
|------|------|
| `method` | HTTP 方法（默认 **GET**） |
| `url` | 请求 URL（必填） |
| `headers` | 可选请求头 |
| `body` | 可选 body；未设 `Content-Type` 时默认 `application/json` |
| `expect_status` | 允许的状态码；省略则要求 **2xx** |
| `insecure_tls` | 跳过 TLS 校验（仅测试环境） |

响应体截断上限为 `admin.jobs.command_max_output_bytes`（默认 **65536**）。

## `script` 任务与安全策略

脚本任务默认允许创建与执行；可通过 `admin.jobs.allow_command: false` 关闭。`kind: command` 仍可作为 **`script` 的兼容别名** 读取。

```yaml
jobs:
  items:
    - id: nightly-backup
      kind: script
      schedule: "0 2 * * *"
      params:
        command: /usr/bin/rsync
        args: ["-a", "/data", "/backup"]
```

```yaml
admin:
  jobs:
    allow_command: true
    command_allowlist:
      - /usr/bin/rsync
      - /bin/echo
    command_workdir: /var/lib/ingress/jobs
    command_max_output_bytes: 65536
```

| 策略字段 | 说明 |
|----------|------|
| `allow_command` | 为 `false` 时禁止配置或执行 `script` 任务 |
| `command_allowlist` | 非空时 **Shell 可执行文件路径** 须与白名单完全一致 |
| `command_workdir` | `params.workdir` 为空时的默认工作目录 |
| `command_max_output_bytes` | stdout+stderr 捕获上限（默认 **65536**） |

脚本参数示例：

```yaml
params:
  engine: shell
  shell: sh
  script: |
    #!/bin/sh
    echo hello
  workdir: /tmp
  env:
    TZ: UTC
```

**脚本类型**

| `engine` | 运行时 | 说明 |
|----------|--------|------|
| `shell`（默认） | 系统 Shell | `shell` 默认 `sh`；`command_allowlist` 限制 Shell 路径 |
| `javascript` | 内置 **goja** | `console.log` / `await fetch(url)`；输出写入执行日志 |
| `go` | 内置 **yaegi** | Go 标准库（`fmt`、`strings`、`time`、`encoding/json`、`net/http` 等）；输出请用 `fmt.Println` |

`shell` 默认为 **`sh`**（解析为 `/bin/sh`）。旧配置仍可使用 `params.command` / `params.args`，保存时会迁移为 `script`。

**`command_allowlist`** 仅对 **`engine: shell`** 生效；**`javascript`** / **`go`** 在进程内解释执行，不走 Shell 白名单。

### Shell（`engine: shell`）

通过 `params.shell`（默认 **`sh`** → `/bin/sh`）以 **`shell -c`** 执行 `params.script`。简单输出请用 **`echo`** 等 shell 内置命令；stdout / stderr 写入任务执行日志。

```yaml
params:
  engine: shell
  shell: sh
  script: |
    #!/bin/sh
    echo "job started"
    date -u
  workdir: /tmp
  env:
    TZ: UTC
```

当 `admin.jobs.command_allowlist` 非空时，解析后的 Shell 路径（如 `/bin/sh`）须在白名单中。

### JavaScript（`engine: javascript`）

进程内 **goja** 解释器。内置 API：

| API | 说明 |
|-----|------|
| `console.log` / `console.error` / `console.warn` | 写入任务日志 |
| `fetch(url, { method, body })` | HTTP 客户端；返回 `{ status, ok, headers, text(), json() }` |

脚本可使用顶层 `await`。示例：

```yaml
params:
  engine: javascript
  script: |
    console.log("job started", new Date().toISOString())
    const res = await fetch("https://backend.internal/healthz")
    console.log("status", res.status, res.ok)
```

### Go（`engine: go`）

进程内 **yaegi** + Go 标准库（`fmt`、`strings`、`strconv`、`time`、`encoding/json`、`os`、`net/http`、`bytes`、`errors` 等）。请用 **`fmt.Println`** / **`fmt.Printf`** 输出（捕获 stdout）。

`import` 写在脚本顶部，其余语句在生成的包装函数中执行：

```yaml
params:
  engine: go
  script: |
    import (
      "fmt"
      "strings"
      "time"
    )

    fmt.Println(strings.ToUpper("job started"), time.Now().Format(time.RFC3339))
```

`engine` 为 `javascript` 或 `go` 时不可设置 `params.shell`。

校验在 YAML 加载及 Admin API 创建/更新时执行；reload 时未通过校验的自定义任务会被 **跳过** 并打警告日志（`jobs: skip custom …`）。

## Admin 界面

访问 **`http://<admin>:<port>/jobs`**（侧栏 **定时任务**）。

- **内置运维** — 开关、改 cron / 参数（如 `retain_days`）、立即执行、查看该任务历史。
- **自定义任务** — 创建 `http_call` / `script`（受策略限制）、编辑、删除、立即执行。
- **执行历史** — 全局最近记录；展开可查看 HTTP 详情或脚本输出摘要。

`GET /api/v1/jobs/capabilities` 反映 `admin.jobs`：`http_call` 始终可用；`script` 依赖 `allow_command`（响应字段仍为 `command`）。

## HTTP API

基础路径：**`/api/v1`**（与 Admin 其他接口相同，无内置认证，请限制网络访问）。

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/jobs` | 列出内置 + 自定义（含 `last_run`） |
| `GET` | `/jobs/capabilities` | 允许的自定义类型 |
| `GET` | `/jobs/runs` | 最近执行（`?job_id=`、`?limit=`） |
| `GET` | `/jobs/runs/:id` | 单条执行（含完整 `result`） |
| `GET` | `/jobs/:source/:id/runs` | 某任务历史（`source`：`builtin` \| `config`） |
| `PUT` | `/jobs/builtins/:id` | 更新内置覆盖并写回 YAML |
| `POST` | `/jobs/items` | 新增自定义任务 |
| `PUT` | `/jobs/items/:id` | 更新（不可改 `kind`） |
| `DELETE` | `/jobs/items/:id` | 删除自定义任务 |
| `POST` | `/jobs/:source/:id/run` | 立即执行（`trigger: manual`） |

通过 API 保存会合并 **`jobs`** 模块到 `ingress.yaml`、校验全文件、落盘并 **`jobs.Reload()`** 刷新 cron。

## 重载行为

调度器在以下情况重载：

1. **`POST /api/v1/reload`** 或 **`POST /api/v1/config/publish`** 在校验通过后（代理 reload + `jobs.Reload()`）。
2. 任意会改写 `ingress.yaml` 的 jobs API（内置/自定义 CRUD）。

`Reload()` 会清空已注册 cron，再注册 **已启用** 且校验通过的内置与自定义任务。未启用或无效项不会入队。

同一任务并发执行会被拒绝（`job "…" is already running`）。

## `job_run` 历史（SQLite）

每次执行在 **`job_run`** 表插入一行（随 Admin 模型自动迁移）：

| 列 | 说明 |
|----|------|
| `job_id` | 任务 ID |
| `source` | `builtin` 或 `config` |
| `kind` | 如 `http_call`、`purge_waf_events` |
| `status` | `running` / `success` / `failed` |
| `trigger` | `schedule` 或 `manual` |
| `duration_ms` | 耗时 |
| `output_preview` | 短摘要（如 `HTTP 200`） |
| `result_detail` | JSON 详情（HTTP 或命令日志） |
| `error` | 失败时的错误信息 |
| `started_at` / `finished_at` | 时间戳 |

成功与失败均会写入审计，动作为 **`job_run`**。

## 快速体验

```bash
ingress run -c examples/jobs/ingress.yaml
# Admin：http://127.0.0.1:9080/jobs
```

手动触发示例：

```bash
curl -sS -X POST http://127.0.0.1:9080/api/v1/jobs/builtin/purge_waf_events/run
curl -sS http://127.0.0.1:9080/api/v1/jobs/runs?limit=10
```

## 示例

可运行配置：[`examples/jobs/`](https://github.com/go-zoox/ingress/tree/master/examples/jobs)，详见 [定时任务示例](../examples/jobs.md)。脚本引擎示例：[`script-engines.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/jobs/script-engines.yaml)。

```bash
ingress validate -c examples/jobs/ingress.yaml
ingress validate -c examples/jobs/http-call-only.yaml
ingress validate -c examples/jobs/builtin-ops.yaml
ingress validate -c examples/jobs/script-engines.yaml
```

另见 [Admin 控制台](./admin.md)（数据库、reload、安全说明）。
