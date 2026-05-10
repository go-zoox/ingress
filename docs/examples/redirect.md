# Redirect examples

`backend.redirect` and `backend.service` cannot both be set on the **same** backend. Redirect-only backends need no `service` block.

For global HTTP→HTTPS (before routing), use `https.redirect_from_http` — see [SSL/TLS](./ssl).

Sources: [`examples/redirect/`](https://github.com/go-zoox/ingress/tree/master/examples/redirect) and [`examples/ssl-tls/route-redirect.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/ssl-tls/route-redirect.yaml) for a minimal host redirect.

## Regex host with captures in `redirect.url`

Same templating as `service.name`: `$1`, `${host.1}`, etc.

The sample below also shows host-level redirect with path backends, and path-only redirect using `${path.N}`:

<<< @/../examples/redirect/capture-and-mixed.yaml yaml

### What each rule demonstrates

1. **Regex host**: `^bigscreen-([^.]+)\.example\.com$` → `redirect.url` uses `$1`.
2. **Host fallback + path services**: default traffic redirects; paths matching `^/api/` proxy to a service.
3. **`${path.N}`**: path regex capture feeds `redirect.url`.
