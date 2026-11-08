# Runtime ban list

Sources live under [`examples/waf-ban/`](https://github.com/go-zoox/ingress/tree/master/examples/waf-ban).

Bans are **operational state**, not configuration. They are created from the Admin Console or the
`/api/v1/waf/bans` API and stored in the admin database — nothing is written into `ingress.yaml`.

## Config (enables the console and trusted proxies)

<<< @/../examples/waf-ban/ingress.yaml

## Create, preview, and remove bans

```bash
# Preview impact before banning
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans/preview \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ip","value":"203.0.113.42"}'

# 24-hour IP ban
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ip","value":"203.0.113.42","ttl_seconds":86400,"reason":"port scan"}'

# User-Agent ban (Go regexp syntax)
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ua_regex","value":"(?i)(nikto|sqlmap)","ttl_seconds":604800}'

# Unban — access is restored immediately
curl -s -X DELETE http://127.0.0.1:9080/api/v1/waf/bans/1
```

## Validate

```bash
ingress validate -c examples/waf-ban/ingress.yaml
```

See [WAF guide](../guide/waf.md#runtime-ban-list) for semantics, guardrails, and limitations.
