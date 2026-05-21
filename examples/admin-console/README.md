# Admin console example bundle

Runnable ingress config, log files, and SQLite-backed admin state for `ingress admin`.

| File | Purpose |
|------|---------|
| `ingress.yaml` | Multi-route sample + **fallback** + **https.ssl (8 certs)** + **cache** (global Redis + route rules) |
| `admin.yaml` | Admin server: points at `ingress.yaml`; log paths default to `/var/log/ingress/*.log` |
| `certs/` | 8 sample TLS certificates (regenerate: `go run ./examples/admin-console/scripts/gen_sample_certs/main.go`) |
| `access.log` | Sample file (~4200 lines) — copy to `/var/log/ingress/access.log` for instant admin metrics, or generate traffic via `ingress run` |
| `error.log` | ~220 lines over the same period |
| `admin.db` | Created on first `ingress admin` start; empty DB gets **180 WAF events**, audit log, config revisions (see `bootstrap/sample.go`) |

Regenerate log files:

```bash
python3 examples/admin-console/scripts/gen_sample_data.py
```

Regenerate TLS certs:

```bash
go run ./examples/admin-console/scripts/gen_sample_certs/main.go
```

```bash
ingress validate -c examples/admin-console/ingress.yaml

# Optional: seed default log dir with bundled sample logs
sudo mkdir -p /var/log/ingress
sudo cp examples/admin-console/access.log /var/log/ingress/
sudo cp examples/admin-console/error.log /var/log/ingress/

# Terminal 1: ingress (logging.enable → /var/log/ingress/*.log; dir auto-created)
ingress run -c examples/admin-console/ingress.yaml

# Terminal 2: admin UI (same default log paths when ingress.log_path is omitted)
ingress admin -c examples/admin-console/admin.yaml
```

`logging.enable: true` without custom `transports` uses **console +** `/var/log/ingress/access.log` and `error.log`. Admin omits `ingress.log_path` / `error_log_path` to follow the same defaults (or paths from `ingress.yaml` when set there).

Reload (SIGHUP) requires **Terminal 1** still running and matching `config_path` / `pid_file`.

**Note:** Delete `admin.db` to re-run bootstrap seed (WAF events + audit log). Log pages always read from configured files only — no in-memory demo fallback.
