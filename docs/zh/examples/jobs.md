# 定时任务示例

源码位于 [`examples/jobs/`](https://github.com/go-zoox/ingress/tree/master/examples/jobs)。

## 最小示例（策略 + http_call + 内置覆盖）

需要 **`admin.enabled`**、SQLite DSN；演示 **`admin.jobs`** 命令策略、一条 **`http_call`** 自定义任务，以及对 **`purge_waf_events`** 的内置覆盖。

<<< @/../examples/jobs/ingress.yaml

```bash
ingress run -c examples/jobs/ingress.yaml
# Admin 定时任务页：http://127.0.0.1:9080/jobs
```

## 仅 HTTP 调用

自定义 **`http_call`** 任务（`expect_status`、请求头、POST body）。无需 `admin.jobs.allow_command`。

<<< @/../examples/jobs/http-call-only.yaml

## 脚本引擎（Shell / JavaScript / Go）

可运行示例：[`examples/jobs/script-engines.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/jobs/script-engines.yaml)，语义见 [定时任务指南](../guide/jobs.md)。

<<< @/../examples/jobs/script-engines.yaml

```bash
ingress validate -c examples/jobs/script-engines.yaml
ingress run -c examples/jobs/script-engines.yaml
# Admin → 定时任务 → 对 shell-echo / js-http-probe / go-stdlib-report 立即执行
```

## 内置运维任务覆盖

调整全部四个内置任务（`purge_waf_events`、`purge_audit_logs`、`check_tls_expiry`、`sync_geoip`）。可选 **`admin.geoip`** 供 GeoIP 同步使用。

<<< @/../examples/jobs/builtin-ops.yaml

## 校验

```bash
ingress validate -c examples/jobs/ingress.yaml
ingress validate -c examples/jobs/http-call-only.yaml
ingress validate -c examples/jobs/builtin-ops.yaml
ingress validate -c examples/jobs/script-engines.yaml
```

语义与 API 见 [定时任务指南](../guide/jobs.md)（cron、`command` 安全、`job_run` 历史等）。
