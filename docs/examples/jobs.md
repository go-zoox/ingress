# Scheduled jobs examples

Sources live under [`examples/jobs/`](https://github.com/go-zoox/ingress/tree/master/examples/jobs).

## Minimal demo (policy + http_call + builtin override)

Requires **`admin.enabled`**, SQLite DSN, and optional **`admin.jobs`** for command policy. Includes one **`http_call`** item and a **`purge_waf_events`** built-in override.

<<< @/../examples/jobs/ingress.yaml

```bash
ingress run -c examples/jobs/ingress.yaml
# Admin jobs UI: http://127.0.0.1:9080/jobs
```

## HTTP call only

Custom **`http_call`** jobs with `expect_status`, headers, and POST body. No `admin.jobs.allow_command` required.

<<< @/../examples/jobs/http-call-only.yaml

## Script engines (shell / JavaScript / Go)

Runnable sample: [`examples/jobs/script-engines.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/jobs/script-engines.yaml) — see [Scheduled jobs guide](../guide/jobs.md#script-engines-shell--javascript--go).

<<< @/../examples/jobs/script-engines.yaml

```bash
ingress validate -c examples/jobs/script-engines.yaml
ingress run -c examples/jobs/script-engines.yaml
# Admin UI → 定时任务 → Run now on shell-echo / js-http-probe / go-stdlib-report
```

## Built-in ops overrides

Tune all four built-in jobs (`purge_waf_events`, `purge_audit_logs`, `check_tls_expiry`, `sync_geoip`). Optional **`admin.geoip`** supports the GeoIP sync job.

<<< @/../examples/jobs/builtin-ops.yaml

## Validate

```bash
ingress validate -c examples/jobs/ingress.yaml
ingress validate -c examples/jobs/http-call-only.yaml
ingress validate -c examples/jobs/builtin-ops.yaml
ingress validate -c examples/jobs/script-engines.yaml
```

See [Scheduled jobs guide](../guide/jobs.md) for cron syntax, command security, API paths, and `job_run` history.
