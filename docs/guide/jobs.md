# Scheduled jobs

Ingress can run **cron-scheduled tasks** inside the same process as the Admin console. Jobs are useful for housekeeping (purge old WAF rows, rotate audit logs), TLS monitoring, GeoIP refresh, and small HTTP or shell integrations without a separate scheduler.

Scheduling requires **`admin.enabled: true`** and a working **`admin.database`** (SQLite by default). Each run is recorded in the **`job_run`** table; the UI is under **维护 → 定时任务** (`/jobs`).

## Built-in vs custom

| Source | Config | Admin UI | Notes |
|--------|--------|----------|-------|
| **Built-in** (`source: builtin`) | Optional `jobs.builtins.<id>` overrides | Editable schedule / enabled / params; not deletable | Registered in code (`core/admin/service/jobs/registry.go`) |
| **Custom** (`source: config`) | `jobs.items[]` in `ingress.yaml` | Full CRUD via API or UI | `kind`: `http_call` or `script` (`command` is a legacy alias) |

### Built-in jobs

| ID | Default schedule | Purpose |
|----|------------------|---------|
| `purge_waf_events` | `0 3 * * *` | Delete WAF events older than `params.retain_days` (default **30**) |
| `purge_audit_logs` | `0 4 * * 0` | Delete admin audit rows older than `params.retain_days` (default **90**) |
| `check_tls_expiry` | `0 */6 * * *` | Scan configured TLS certs; warn on expiring/expired |
| `sync_geoip` | `0 2 * * *` | Reload GeoIP settings from `ingress.yaml` (`admin.geoip`) |

Override example:

```yaml
jobs:
  builtins:
    purge_waf_events:
      enabled: true
      schedule: "0 3 * * *"
      params:
        retain_days: 14
```

Omit `jobs.builtins` entirely to use each built-in’s code defaults (all enabled unless you disable one with `enabled: false`).

### Custom jobs (`jobs.items[]`)

```yaml
jobs:
  items:
    - id: nightly-health
      name: Nightly health probe
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

| Field | Description | Default |
|-------|-------------|---------|
| `id` | Unique job id (required) | — |
| `name` | Display name | `id` |
| `kind` | `http_call` or `script` (`command` is a legacy alias) | — |
| `schedule` | Cron expression (5 fields) | required |
| `enabled` | Register with cron when `true` | `false` if omitted in YAML decode |
| `timeout_sec` | Per-run timeout | **60** |
| `on_failure` | `log`, `retry`, or `disable` | `log` |
| `params` | Kind-specific (see below) | — |

**`on_failure` behavior**

- **`log`** — Record failure in `job_run` and an audit row (`job_run`); next run waits for the cron tick.
- **`disable`** — For **custom** jobs only: set `enabled: false` in `ingress.yaml`, save, and reload the job scheduler.
- **`retry`** — Accepted in config; there is no immediate re-run—failures still wait for the next cron schedule.

## Cron expressions

Schedules use the zoox cron component (standard **5-field** cron: `minute hour day month weekday`). Examples:

| Expression | Meaning |
|------------|---------|
| `0 3 * * *` | Daily at 03:00 |
| `0 4 * * 0` | Sundays at 04:00 |
| `*/15 * * * *` | Every 15 minutes |
| `0 */6 * * *` | Every 6 hours |

Invalid or empty schedules are rejected when saving custom jobs; built-in overrides with an empty `schedule` keep the built-in default.

## `http_call` jobs

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

| Param | Description |
|-------|-------------|
| `method` | HTTP method (default **GET**) |
| `url` | Request URL (required) |
| `headers` | Optional request headers |
| `body` | Optional body; sets `Content-Type: application/json` if missing |
| `expect_status` | Allowed status codes; if omitted, **2xx** is required |
| `insecure_tls` | Skip TLS verification (use only in lab) |

Response bodies are truncated using `admin.jobs.command_max_output_bytes` (default **65536**).

## `script` jobs and security

Script jobs are enabled by default; set `admin.jobs.allow_command: false` to disable them. `kind: command` is still accepted as a **legacy alias** for `script`.

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

| Policy field | Description |
|--------------|-------------|
| `allow_command` | When `false`, script jobs cannot be defined or run |
| `command_allowlist` | If non-empty, the resolved **Shell path** must match an entry exactly |
| `command_workdir` | Default working directory when `params.workdir` is empty |
| `command_max_output_bytes` | Max captured stdout+stderr (default **65536**) |

Script item params:

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

| `engine` | Runtime | Notes |
|----------|---------|-------|
| `shell` (default) | Host shell | `shell` defaults to `sh`; `command_allowlist` applies to the shell binary |
| `javascript` | Embedded **goja** | `console.log`, `await fetch(url)` |
| `go` | Embedded **yaegi** | Go stdlib via yaegi (`fmt`, `strings`, `time`, `encoding/json`, `net/http`, …); use `fmt.Println` for output |

Legacy `params.command` / `params.args` are migrated to `script` on save. **`command_allowlist`** applies only to **`engine: shell`**. Embedded **`javascript`** and **`go`** engines run in-process and ignore the allowlist.

### Shell (`engine: shell`)

Runs `params.script` via `params.shell` (default **`sh`** → `/bin/sh`) with **`shell -c`**. Use shell builtins such as **`echo`** for simple output; stdout and stderr are captured in the job run log.

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

When `admin.jobs.command_allowlist` is non-empty, the resolved shell binary (e.g. `/bin/sh`) must appear in the list.

### JavaScript (`engine: javascript`)

Runs in-process with **goja**. Built-ins:

| API | Description |
|-----|-------------|
| `console.log` / `console.error` / `console.warn` | Append lines to the job log |
| `fetch(url, { method, body })` | HTTP client; returns `{ status, ok, headers, text(), json() }` |

Scripts may use top-level `await`. Example:

```yaml
params:
  engine: javascript
  script: |
    console.log("job started", new Date().toISOString())
    const res = await fetch("https://backend.internal/healthz")
    console.log("status", res.status, res.ok)
```

### Go (`engine: go`)

Runs in-process with **yaegi** and the Go standard library (`fmt`, `strings`, `strconv`, `time`, `encoding/json`, `os`, `net/http`, `bytes`, `errors`, …). Write output with **`fmt.Println`** / **`fmt.Printf`** (stdout is captured).

Place `import` lines at the top of the script; remaining statements run inside a generated wrapper function:

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

`params.shell` is not allowed when `engine` is `javascript` or `go`.

Legacy `params.command` / `params.args` are migrated to `script` on save.

Validation runs when items are created/updated in YAML or via the Admin API. Jobs that fail validation at reload are **skipped** with a warning log (`jobs: skip custom …`).

## Admin UI

Open **`http://<admin-host>:<admin.port>/jobs`** (menu **定时任务**).

- **Built-in ops** — Toggle enabled, edit cron and params (e.g. `retain_days`), run now, view per-job history.
- **Custom jobs** — Create `http_call` or `script` items (when allowed), edit, delete, run now.
- **History** — Recent runs across all jobs; expand a run for HTTP status/body or script log preview.

Capabilities (`GET /api/v1/jobs/capabilities`) reflect `admin.jobs`: `http_call` is always available; script jobs require `allow_command` (JSON field remains `command`).

## HTTP API

Base path: **`/api/v1`** (same auth posture as the rest of the admin API—restrict network access).

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/jobs` | List builtins + custom items (`last_run` when present) |
| `GET` | `/jobs/capabilities` | Which custom kinds are allowed |
| `GET` | `/jobs/runs` | Recent runs (`?job_id=`, `?limit=`) |
| `GET` | `/jobs/runs/:id` | One run with full `result` payload |
| `GET` | `/jobs/:source/:id/runs` | Runs for one job (`source`: `builtin` \| `config`) |
| `PUT` | `/jobs/builtins/:id` | Patch built-in override in `ingress.yaml` |
| `POST` | `/jobs/items` | Add custom item |
| `PUT` | `/jobs/items/:id` | Update custom item (`kind` cannot change) |
| `DELETE` | `/jobs/items/:id` | Remove custom item |
| `POST` | `/jobs/:source/:id/run` | Run now (`trigger: manual`) |

Saving builtins or items through the API merges the **`jobs`** module into `ingress.yaml`, validates the full file, writes disk, and calls **`jobs.Reload()`** to refresh cron entries.

## Reload behavior

The job scheduler reloads when:

1. **`POST /api/v1/reload`** or **`POST /api/v1/config/publish`** after ingress validates (proxy reload + `jobs.Reload()`).
2. Any jobs API write that updates `ingress.yaml` (builtin/custom CRUD).

`Reload()` clears all cron registrations and re-adds **enabled** built-ins and **enabled** custom items that pass validation. Disabled or invalid jobs are not scheduled.

Concurrent runs of the same job are rejected (`job "…" is already running`).

## `job_run` history (SQLite)

Each execution creates a row in **`job_run`** (auto-migrated with other admin models):

| Column | Description |
|--------|-------------|
| `job_id` | Job identifier |
| `source` | `builtin` or `config` |
| `kind` | e.g. `http_call`, `purge_waf_events` |
| `status` | `running`, `success`, `failed` |
| `trigger` | `schedule` or `manual` |
| `duration_ms` | Wall time |
| `output_preview` | Short summary (e.g. `HTTP 200`) |
| `result_detail` | JSON detail (HTTP body/headers or command log) |
| `error` | Error message when failed |
| `started_at` / `finished_at` | Timestamps |

Successful and failed runs also append an admin **audit** event with action **`job_run`**.

## Quick start

```bash
ingress run -c examples/jobs/ingress.yaml
# Admin UI: http://127.0.0.1:9080/jobs
```

Manual run (example):

```bash
curl -sS -X POST http://127.0.0.1:9080/api/v1/jobs/builtin/purge_waf_events/run
curl -sS http://127.0.0.1:9080/api/v1/jobs/runs?limit=10
```

## Examples

Runnable samples: [`examples/jobs/`](https://github.com/go-zoox/ingress/tree/master/examples/jobs) — see [Scheduled jobs examples](../examples/jobs.md). Script engines: [`script-engines.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/jobs/script-engines.yaml).

```bash
ingress validate -c examples/jobs/ingress.yaml
ingress validate -c examples/jobs/http-call-only.yaml
ingress validate -c examples/jobs/builtin-ops.yaml
ingress validate -c examples/jobs/script-engines.yaml
```

See also [Admin console](./admin.md) for database, reload, and security notes.
