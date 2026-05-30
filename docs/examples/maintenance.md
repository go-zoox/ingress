# Maintenance examples

Sources live under [`examples/maintenance/`](https://github.com/go-zoox/ingress/tree/master/examples/maintenance).

## Global always-on (status probe)

Hosts in `maintenance.hosts` with **no `window`** are active whenever the hostname matches. Use **`GET /_/ingress/status`** to probe maintenance state for a Host (returns `{"status":"ok"}` or `{"status":"maintenance",...}`).

<<< @/../examples/maintenance/global-always-on.yaml

```bash
curl -sS http://app.example.com/_/ingress/status
curl -sS -D - http://app.example.com/api   # 503 + X-Ingress-Maintenance: true when active
```

## Global maintenance + bypass

<<< @/../examples/maintenance/global-bypass.yaml

While maintenance is active, **`/healthz`** (handler) and requests with **`X-Maintenance-Bypass: secret-token`** pass through; other paths receive 503.

## Route `scope: all`

<<< @/../examples/maintenance/route-scope-all.yaml

## Route `scope: listed` + per-host windows

<<< @/../examples/maintenance/route-scope-listed.yaml

## Custom maintenance response header

<<< @/../examples/maintenance/custom-response-header.yaml

## Custom status probe path

<<< @/../examples/maintenance/custom-status-path.yaml

## Global + route-level combined

<<< @/../examples/maintenance/ingress.yaml

## Validate

```bash
ingress validate -c examples/maintenance/global-always-on.yaml
ingress validate -c examples/maintenance/global-bypass.yaml
ingress validate -c examples/maintenance/route-scope-all.yaml
ingress validate -c examples/maintenance/route-scope-listed.yaml
ingress validate -c examples/maintenance/custom-response-header.yaml
ingress validate -c examples/maintenance/custom-status-path.yaml
ingress validate -c examples/maintenance/ingress.yaml
```

See [Maintenance guide](../guide/maintenance.md) for semantics, response headers, and access-log fields.
