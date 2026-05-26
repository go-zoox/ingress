# Admin Console

Ingress can embed an **operations console** in the same process as the reverse proxy. Enable it with the top-level **`admin:`** block in `ingress.yaml` — there is no separate `ingress admin` subcommand.

The console exposes an HTTP API (default port **9080**) and a React UI for routes, logs, TLS, cache, WAF events, and config editing with validate / publish / reload.

## Quick start

Minimal config:

```yaml
version: v1
port: 8080

admin:
  enabled: true
  port: 9080

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

Run:

```bash
ingress run -c ingress.yaml
```

Startup logs include:

```text
Admin started at http://127.0.0.1:9080
Server started at http://127.0.0.1:8080
```

Open **http://127.0.0.1:9080** for the built-in UI (when `admin.web.dev_proxy` is `false`, the default in production builds).

Full demo bundle: [`examples/admin-console/`](https://github.com/go-zoox/ingress/tree/master/examples/admin-console) — multi-route sample, sample logs, TLS certs, and SQLite state. See the [Admin console example](/examples/admin-console).

## Configuration

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `admin.enabled` | bool | Start the admin server with `ingress run` | `false` |
| `admin.port` | int | Admin listen port | `9080` |
| `admin.database.driver` | string | Audit / revision DB driver | `sqlite` |
| `admin.database.dsn` | string | SQLite DSN (relative paths resolve beside the ingress config file) | `file:admin.db?cache=shared&_fk=1` |
| `admin.web.dev_proxy` | bool | API only; run the UI from Vite dev server (proxies `/api`) | `false` |
| `admin.access_log_path` | string | Override access log path for the log viewer | from `logging` file transport |
| `admin.error_log_path` | string | Override error log path for the log viewer | from `logging` file transport |

Example with local SQLite and UI dev mode:

<<< @/../examples/admin-console/ingress.yaml{yaml}

Only the **`admin:`** stanza is required for the console; the rest of the file is a routing demo.

## Logging and the log viewer

The admin **Logs** page reads from file paths on disk. By default it uses the same paths as ingress **`logging`** (after prepare/normalize).

When **`admin.enabled: true`** and **`logging` is unset** (no `enable`, `level`, or `transports`), ingress defaults to:

- `logging.enable: true`
- File transport beside the config file: `access.log` and `error.log`

**Explicit `logging.*` always wins** — including `logging.enable: false` or custom `transports`. When admin is **disabled**, unset logging still defaults to `/var/log/ingress/access.log` and `error.log` if you set `logging.enable: true` without transports.

Override only for the admin reader (without changing ingress logging):

```yaml
admin:
  enabled: true
  access_log_path: /var/log/ingress/access.log
  error_log_path: /var/log/ingress/error.log
```

Access log line format matches [Configuration · Access log fields](/guide/configuration#access-log-fields). Query filters include `cache_hit`, `waf_block`, `host`, `status`, and byte `offset` for tailing.

## UI development

For frontend work, enable dev proxy and run Vite separately:

```yaml
admin:
  enabled: true
  web:
    dev_proxy: true
```

```bash
ingress run -c ingress.yaml
cd core/admin/web && pnpm dev
```

The Vite dev server proxies `/api` to the admin port. Production UI is embedded at compile time after `cd core/admin && make build` (`-tags adminui`; output under `core/admin/static/dist`, not committed to git).

## HTTP API

Base path: **`/api/v1`**. Responses use JSON envelopes from the admin handler layer.

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/status` | Process / config summary |
| `GET` | `/routes` | Flattened route table |
| `POST` | `/routes/match` | Dry-run match (`host`, `path` JSON body) |
| `GET` | `/logs` | Tail/search access or error logs |
| `GET` | `/metrics/overview` | Aggregates from access log window |
| `GET` | `/waf/events` | Recent WAF audit rows (SQLite) |
| `GET` | `/tls/certs` | Certificate metadata from config paths |
| `POST` | `/tls/certs/check` | Inspect one domain |
| `GET` | `/cache/overview` | Cache engine / key overview |
| `GET` | `/config` | Read ingress YAML |
| `PUT` | `/config` | Save YAML (revision recorded) |
| `POST` | `/config/validate` | Validate YAML body or on-disk file |
| `POST` | `/config/preview` | Diff / preview pending changes |
| `POST` | `/config/publish` | Validate, save, and reload |
| `POST` | `/config/modules` | List config editor modules |
| `POST` | `/config/modules/merge` | Merge one module patch |
| `GET` | `/config/revisions` | List saved revisions |
| `GET` | `/config/revisions/:id` | One revision |
| `POST` | `/reload` | Validate on-disk config and reload ingress |
| `GET` | `/routes/:ri/:pi` | Route detail (config + auth/cache/healthcheck) |
| `GET` | `/routes/:ri/:pi/metrics` | Route-level aggregated metrics |
| `GET` | `/events/stream` | SSE real-time event stream (`?channels=...`) |
| `GET` | `/healthcheck` | Health check probe results and summary |
| `GET` | `/settings` | Admin + ingress settings snapshot |

**Reload from the console** validates the config file, then triggers an in-process reload (same outcome as **`ingress reload`** / **SIGHUP** when started with `ingress run`).

## Real-time events (SSE)

The admin console pushes real-time updates over **Server-Sent Events** (SSE). Connect to:

```
GET /api/v1/events/stream?channels=metrics,waf,logs,health
```

Supported channels: `metrics`, `waf`, `logs`, `health`. The UI auto-subscribes based on the current page. Up to **5 concurrent SSE connections per IP** are allowed; the client automatically falls back to polling if the limit is reached or SSE is unavailable.

Event format:

```
event: channel:action
data: {"key": "value", ...}
```

## Route detail

Click any route row to open the **route detail** page at `/routes/:ruleIndex/:pathIndex`. It shows:

- Full route configuration (host, path, backend, auth, cache, health check, WAF)
- Real-time metrics (QPS, latency percentiles, error rate, cache hit ratio)
- Filtered logs, WAF events, and cache data for that specific route

## Topology graph

The **Topology** page (`/topology`) renders a three-layer SVG diagram: **Host → Path → Backend**. Node colors indicate health status (green = up, yellow = warning, red = down). Click a node to navigate to the route detail or routes list.

## Health check panel

The **Health** page (`/health`) displays the status of all backends that have `healthcheck` configured. The backend probes every **30 seconds** with a **5-second timeout**. State changes are pushed via the SSE `health` channel.

## Config draft & undo

The config editor tracks edit history (up to **50 steps**). Use **Ctrl+Z** / **Cmd+Z** to undo, **Ctrl+Shift+Z** / **Cmd+Shift+Z** to redo. The draft badge in the tab bar shows unsaved changes.

## One-click rollback

In the **config version history** panel, each revision has a **Rollback** button. Clicking it opens a confirmation dialog, then validates, publishes, and reloads the selected version — no manual copy-paste needed.

## Certificate expiry alerts

The **Overview** page now reads real TLS certificate data instead of hard-coded values. Certificates expiring within **30 days** show a yellow warning; within **7 days**, a red critical alert.

## Version consistency badge

The Overview page compares the **running config hash** with the **latest revision hash**. A green badge means "config consistent"; yellow means "changes not yet published".

## Security notes

- The admin API has **no built-in authentication** in v1. Bind to localhost or place it behind a trusted network, VPN, or ingress route with auth.
- Config publish writes the live `ingress.yaml` and reloads the proxy — restrict who can reach the admin port.
- Do not expose **`admin.web.dev_proxy: true`** on untrusted networks.

## Related commands

Validate before run or deploy:

```bash
ingress validate -c ingress.yaml
```

Reload after editing the file on disk:

```bash
ingress reload -c ingress.yaml
# or: kill -HUP $(cat /tmp/gozoox.ingress.pid)
```

See [Getting started · Command line options](/guide/getting-started#command-line-options) for `run`, `validate`, and `reload` flags.
