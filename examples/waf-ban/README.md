# Runtime ban list

Bans are **operational state**, not ingress configuration. They are created from the Admin Console
(**安全 → 封禁名单**, or the 封禁 IP / 封禁 UA buttons on any WAF event row) and stored in the admin
database. The running ingress receives them as an immutable compiled snapshot.

## Run it

```bash
# from the repo root
go run -tags adminui ./cmd/ingress run -c examples/waf-ban/ingress.yaml
# Console: http://127.0.0.1:9080  (gateway on 8080)
```

Then create a ban from the console, or through the API:

```bash
# 24-hour IP ban
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ip","value":"203.0.113.42","ttl_seconds":86400,"reason":"port scan"}'

# User-Agent ban (Go regexp syntax)
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ua_regex","value":"(?i)(nikto|sqlmap)","ttl_seconds":604800}'

# Estimate impact before banning
curl -s -X POST http://127.0.0.1:9080/api/v1/waf/bans/preview \
  -H 'Content-Type: application/json' \
  -d '{"kind":"ip","value":"203.0.113.42"}'

# Unban (restores access immediately, no reload)
curl -s -X DELETE http://127.0.0.1:9080/api/v1/waf/bans/1
```

Verify the block and its access-log field:

```bash
curl -s -o /dev/null -w '%{http_code}\n' -H 'Host: api.example.com' http://127.0.0.1:8080/
# 403, and the access log line carries: waf_block=0 ban_block=1
```

## What this example demonstrates

| Behavior | Detail |
|---|---|
| No reload | Creating or removing a ban never touches `ingress.yaml`, never flushes the response cache, never resets rate limiters |
| Survives WAF off | The ban list is checked before WAF and is independent of `waf.enabled` |
| Beats the allow list | A ban takes precedence over `rules[].waf.allow` |
| Works before rate limiting | A banned client gets **403**, not **429** |
| TTL without a job | `expires_at` is evaluated per request; an expired ban stops matching immediately |
| Real client IP | Same resolution as WAF: TCP peer, then `trusted_proxies`; a spoofed `X-Forwarded-For` from an untrusted peer is ignored |

## Guardrails

- Banning an address inside `waf.trusted_proxies` is **rejected** (that is the proxy itself).
- Permanent bans require `confirm_permanent: true`.
- Duplicate active bans (`kind` + `value` + `hosts`) are rejected.

## Limitations

- **Single replica only**: the ban is injected into the ingress instance owned by the admin process that received
  the API call. Multi-replica sync is not implemented yet.
- **UA bans are bypassable**: `User-Agent` is client-controlled. Treat them as a stop-gap.
