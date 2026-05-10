# SSL/TLS Examples

This page provides examples of SSL/TLS configuration.

## Basic HTTPS Configuration

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem
```

## Multiple Domains

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem
    - domain: api.example.com
      cert:
        certificate: /etc/ssl/api.example.com/fullchain.pem
        certificate_key: /etc/ssl/api.example.com/privkey.pem
    - domain: admin.example.com
      cert:
        certificate: /etc/ssl/admin.example.com/fullchain.pem
        certificate_key: /etc/ssl/admin.example.com/privkey.pem
```

## Let's Encrypt Certificates

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/letsencrypt/live/example.com/fullchain.pem
        certificate_key: /etc/letsencrypt/live/example.com/privkey.pem
```

## HTTPS with Backend Services

```yaml
version: v1
port: 8080

https:
  port: 8443
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http  # Backend uses HTTP, TLS terminated at Ingress
```

## Global HTTP to HTTPS redirect

When `https.port` is set, Ingress can force cleartext HTTP clients to HTTPS **before** route matching. Configure `https.redirect_from_http` (not `rules[].backend.redirect`):

```yaml
version: v1
port: 8080

https:
  port: 8443
  redirect_from_http:
    permanent: true
    # with_origin_method_and_body: false   # true -> 308/307, false -> 301/302
    # exclude_paths:
    #   - /healthz
  ssl:
    - domain: example.com
      cert:
        certificate: /etc/ssl/example.com/fullchain.pem
        certificate_key: /etc/ssl/example.com/privkey.pem

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        protocol: http
```

## Route-level redirect (`rules[].backend.redirect`)

Use `backend.redirect` when a **specific host or path** should issue a redirect instead of proxying. **`service` and `redirect` are mutually exclusive** on the same backend; you may omit `service` when using redirect only.

```yaml
rules:
  - host: old.example.com
    backend:
      redirect:
        url: https://new.example.com
        permanent: true
        # with_origin_method_and_body: false
```

## Testing

### HTTPS Request

```bash
curl https://example.com:8443/api
```

### Verify Certificate

```bash
openssl s_client -connect example.com:8443 -servername example.com
```

### Certificate Reload

After updating certificates, reload the configuration:

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```
