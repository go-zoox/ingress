# Forward Proxy (CONNECT + absolute-form)

Ingress can terminate TLS for an **HTTP forward proxy** and forward both standard proxy request forms to a plain-HTTP proxy hop.

- `https://target` → `CONNECT target:443`
- `http://target` → `GET http://target/...` (absolute-form)

The HTTP `Host` is the destination, so ingress matches the route by **TLS SNI**.

## Configuration

```yaml
https:
  port: 443
  ssl:
    - domain: proxy.example.com
      cert:
        certificate: /etc/ssl/proxy.example.com.pem
        certificate_key: /etc/ssl/proxy.example.com.key.pem

rules:
  - host: proxy.example.com
    backend:
      service:
        name: anchor-frontend
        port: 8080
        protocol: http
    connect_proxy:
      enabled: true
      upstream:
        name: anchor-backend
        port: 3128
        protocol: http
      allowed_targets:
        - '^.+:(80|443)$'
      connect_timeout: 10
      response_timeout: 10
      idle_timeout: 600
      max_connections: 500
```

`connect_proxy.upstream` is independent from the normal `backend`. If it is omitted, the rule's `backend.service` is used for backward compatibility.

## Authentication

Ingress forwards the client's `Proxy-Authorization` header to the proxy hop. The downstream proxy (Anchor) validates the account and returns `407 Proxy Authentication Required`; ingress relays that response and `Proxy-Authenticate` unchanged. Ingress does not store proxy passwords.

## Target allowlist

`allowed_targets` is an optional list of full-string regular expressions matched against the canonical `host:port`:

- CONNECT: the authority from the CONNECT request (`example.com:443`).
- HTTP absolute-form: `host:port`, with an implicit `:80` when no port is present.

An empty list allows all targets. A non-matching target returns `403` without contacting the proxy hop.

> Always anchor patterns with `^` and `$`. The same regexp is applied to the complete authority, so `^example\.com:443$` does not match `evil-example.com:443`.

## Limits

- `connect_timeout` — dial and CONNECT write timeout, default 10 seconds.
- `response_timeout` — CONNECT response header timeout, default 10 seconds.
- `idle_timeout` — tunnel idle timeout, default 600 seconds.
- `max_connections` — concurrent proxy requests per route; `0` means unlimited.

Proxy requests also use the rule/global `rate_limit` and are included in access logs.

## Usage

```bash
# HTTPS target (CONNECT)
curl -x https://proxy.example.com \
  -U 'mp_user:password' \
  https://httpbin.org/ip

# HTTP target (absolute-form)
curl -x https://proxy.example.com \
  -U 'mp_user:password' \
  http://httpbin.org/get
```

TLS must terminate at ingress. A separate TLS-terminating load balancer in front of ingress prevents ingress from seeing SNI and CONNECT.

## Admin Console

Open **Config → rules**, edit the rule, and choose **Forward Proxy** in the section navigator. The form edits the dedicated upstream, target ACL, timeouts, and connection limit; normal Backend remains in the **Backend** section. The rule list summary shows `PROXY → upstream`.
