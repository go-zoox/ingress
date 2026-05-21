# Admin console example bundle

Runnable ingress config, log files, and SQLite-backed admin state for `ingress admin`.

| File | Purpose |
|------|---------|
| `ingress.yaml` | Multi-route sample (API, CDN wildcard, regex inlets, handler, redirect, WAF) |
| `admin.yaml` | Admin server: points at `ingress.yaml`, `access.log`, `error.log`, `admin.db` |
| `access.log` | ~4200 lines, **90 days** (Feb–May 2026) — logs API & overview metrics |
| `error.log` | ~220 lines over the same period |
| `admin.db` | Created on first `ingress admin` start; empty DB gets **180 WAF events**, audit log, config revisions (see `bootstrap/sample.go`) |

Regenerate log files:

```bash
python3 examples/admin-console/scripts/gen_sample_data.py
```

```bash
ingress validate -c examples/admin-console/ingress.yaml

# Terminal 1: ingress (writes pid + can append access/error logs when logging is enabled)
ingress run -c examples/admin-console/ingress.yaml

# Terminal 2: admin UI (reads logs from files, WAF events from SQLite)
ingress admin -c examples/admin-console/admin.yaml
```

Paths in `admin.yaml` are resolved relative to **that file's directory**. Keep `logging.transports` in `ingress.yaml` aligned with the same `access.log` / `error.log` paths.

Reload (SIGHUP) requires **Terminal 1** still running and matching `config_path` / `pid_file`.

**Note:** Delete `admin.db` to re-run bootstrap seed (WAF events + audit log). Log pages always read from configured files only — no in-memory demo fallback.
