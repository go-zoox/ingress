# Scheduled jobs examples

Runnable configs for [scheduled jobs](../../docs/guide/jobs.md) (`jobs:` + `admin.jobs`).

| File | Purpose |
|------|---------|
| `ingress.yaml` | Minimal demo: built-in override, `http_call`, shell `script` |
| `http-call-only.yaml` | Custom `http_call` jobs only (no `allow_command`) |
| `script-engines.yaml` | Shell / JavaScript(goja) / Go(yaegi) script engines |
| `builtin-ops.yaml` | Override all four built-in ops jobs |

```bash
ingress validate -c examples/jobs/ingress.yaml
ingress validate -c examples/jobs/script-engines.yaml
ingress run -c examples/jobs/script-engines.yaml
# Admin jobs UI: http://127.0.0.1:9080/jobs
```

Manual run (after `ingress run`):

```bash
curl -sS -X POST http://127.0.0.1:9080/api/v1/jobs/config/shell-echo/run
curl -sS http://127.0.0.1:9080/api/v1/jobs/runs?job_id=shell-echo&limit=5
```
