# Admin Console

Runnable bundle with embedded admin, sample access/error logs, TLS certs, and SQLite-backed audit state.

Source: [`examples/admin-console/`](https://github.com/go-zoox/ingress/tree/master/examples/admin-console).

## Configuration

<<< @/../examples/admin-console/ingress.yaml{yaml}

Key points:

- **`admin.enabled: true`** — API on port **9080** in the same process as the proxy (**8080** / **8443**).
- **`admin.web.dev_proxy: true`** — API only; run `cd core/admin/web && pnpm dev` for the UI.
- When **`logging`** is omitted, file logs default to **`./access.log`** and **`./error.log`** next to this YAML (no `/var/log/ingress` required).

## Validate and run

```bash
ingress validate -c examples/admin-console/ingress.yaml
ingress run -c examples/admin-console/ingress.yaml
```

Expected startup lines:

```text
Admin started at http://127.0.0.1:9080
Server started at http://127.0.0.1:8080
```

## Sample data

| Asset | Purpose |
|-------|---------|
| `access.log` / `error.log` | Pre-generated lines for the Logs UI |
| `admin.db` | Created on first start; empty DB gets bootstrap WAF events and audit rows |
| `certs/` | Sample TLS files referenced by `https.ssl` |

Regenerate helpers (from repo root):

```bash
python3 examples/admin-console/scripts/gen_sample_data.py
go run ./examples/admin-console/scripts/gen_sample_certs/main.go
```

See also the [Admin console guide](/guide/admin).
