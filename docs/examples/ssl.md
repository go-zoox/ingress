# SSL/TLS Examples

This page provides examples of SSL/TLS configuration.

Sources: [`examples/ssl-tls/`](https://github.com/go-zoox/ingress/tree/master/examples/ssl-tls).

## Basic HTTPS configuration

<<< @/../examples/ssl-tls/https-basic.yaml

## Multiple domains

<<< @/../examples/ssl-tls/https-multi-domain.yaml

## Let's Encrypt certificates

<<< @/../examples/ssl-tls/https-letsencrypt.yaml

## HTTPS with backend services

<<< @/../examples/ssl-tls/https-with-backends.yaml

## Global HTTP to HTTPS redirect

When `https.port` is set, Ingress can force cleartext HTTP clients to HTTPS **before** route matching. Configure `https.redirect_from_http` (not `rules[].backend.redirect`):

<<< @/../examples/ssl-tls/https-global-redirect.yaml

Optional fields (comment in your own file as needed):

- `with_origin_method_and_body`: `false` → 301/302 family; `true` → 307/308
- `exclude_paths`: exact paths that skip the forced redirect

## Route-level redirect (`rules[].backend.redirect`)

Use `backend.redirect` when a **specific host or path** should issue a redirect instead of proxying. **Usually omit `backend.type`**—Ingress infers **`redirect`** when only **`redirect`** is configured. **Runnable comparison:** **`examples/ssl-tls/route-redirect.yaml`** uses two hosts (`type: redirect` vs omission). Set **`backend.type: redirect`** explicitly only when validation reports ambiguity. See [routing](/guide/routing#redirects) for how **`service`**, **`handler`**, and **`redirect`** blocks relate to each backend.

<<< @/../examples/ssl-tls/route-redirect.yaml

For regex capture templating in `redirect.url`, see [Redirects](./redirect).

## Testing

### HTTPS request

```bash
curl https://example.com:8443/api
```

### Verify certificate

```bash
openssl s_client -connect example.com:8443 -servername example.com
```

### Certificate reload

After updating certificates, reload the configuration:

```bash
kill -HUP $(cat /tmp/gozoox.ingress.pid)
```
