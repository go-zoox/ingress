# Admin console example bundle

Runnable ingress config with embedded **admin** console, log files, and SQLite-backed admin state.

| File | Purpose |
|------|---------|
| `ingress.yaml` | Multi-route sample + **admin** + **fallback** + **https.ssl (8 certs)** + **cache** |
| `certs/` | 8 sample TLS certificates (regenerate: `go run ./examples/admin-console/scripts/gen_sample_certs/main.go`) |
| `access.log` | Sample access log (~4200 lines) — referenced by `logging.transports` in `ingress.yaml` |
| `error.log` | Sample error log (~220 lines) |
| `admin.db` | Created on first start when `admin.enabled: true`; empty DB gets **180 WAF events**, audit log, config revisions (see `core/admin/bootstrap/sample.go`) |

Regenerate log files:

```bash
python3 examples/admin-console/scripts/gen_sample_data.py
```

Regenerate TLS certs:

```bash
go run ./examples/admin-console/scripts/gen_sample_certs/main.go
```

```bash
# Rebuild after pulling changes (admin is embedded in ingress run)
go build -o ingress ../../cmd/ingress

./ingress validate -c ingress.yaml

# Single process: ingress proxy (8080) + admin API (9080)
./ingress run -c ingress.yaml
```

Startup logs should include:

```text
Admin started at http://127.0.0.1:9080
Server started at http://127.0.0.1:8080
```

`logging` writes to **`./access.log`** and **`./error.log`** next to this config (no `/var/log/ingress` required). Admin follows the same paths when `admin.access_log_path` / `error_log_path` are omitted.

Admin UI dev mode (`admin.web.dev_proxy: true`): run `cd core/admin/web && pnpm dev` and open the Vite dev server (proxies `/api`).

Reload from the admin console applies in-process. External reload still works via SIGHUP when using `ingress reload`.

**Note:** Delete `admin.db` to re-run bootstrap seed (WAF events + audit log). Log pages always read from configured files only — no in-memory demo fallback.
