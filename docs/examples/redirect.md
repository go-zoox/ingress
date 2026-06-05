# Redirect examples

Normally omit **`backend.type`**: Ingress infers **`redirect`** when only **`backend.redirect`** is set. **`examples/ssl-tls/route-redirect.yaml`** pairs **`type: redirect`** with omission on two hosts so you can compare. Add **`backend.type: redirect`** only if **`ingress validate`** reports ambiguity. **`examples/redirect/capture-and-mixed.yaml`** also mixes explicit **`backend.type`** on some backends with omission elsewhere. Do not combine **`backend.service`** or **`backend.handler`** with **`backend.redirect`** on the **same** `backend`—use separate **`paths`** entries when one host needs both proxy and redirect (see below).

For global HTTP→HTTPS (before routing), use `https.redirect_from_http` — see [SSL/TLS](./ssl).

Sources: [`examples/redirect/`](https://github.com/go-zoox/ingress/tree/master/examples/redirect) and [`examples/ssl-tls/route-redirect.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/ssl-tls/route-redirect.yaml) for a minimal host redirect.

## Redirect status (`duration` + `preserve_request`)

Use **`duration`** (`temporary` | `permanent`) and **`preserve_request`** (`true` | `false`) to pick **302 / 301 / 307 / 308** without memorizing status codes. Legacy **`permanent`** / **`with_origin_method_and_body`** still work; prefer the new fields.

| duration | preserve_request | Status |
|----------|------------------|--------|
| temporary | false | 302 |
| permanent | false | 301 |
| temporary | true | 307 |
| permanent | true | 308 |

<<< @/../examples/redirect/redirect-status.yaml

Each host in the sample returns a different status on GET. **`preserve_request: true`** is for POST/API redirects where method and body must survive the redirect.

## Path regex redirect

`paths[].path` is a Go regexp (compiled with an implicit leading `^`). Path-level **`backend.redirect`** uses the same matcher as service/handler paths. Use **`${path.N}`** for capture groups in `redirect.url`; add **`$`** when you need an exact path match.

<<< @/../examples/redirect/path-regex.yaml

Prefix `/legacy/` sends matching traffic to a fixed URL; `/go/([^/]+)$` expands `${path.1}`; `/promo$` matches only `/promo` and redirects to an explicit landing path. Unmatched paths fall back to the host-level redirect (preserving path and query).

## Regex host with captures in `redirect.url`

Same templating as `service.name`: `$1`, `${host.1}`, etc.

The sample below also shows host-level redirect with path backends, and path-only redirect using `${path.N}`:

<<< @/../examples/redirect/capture-and-mixed.yaml

### What each rule demonstrates

1. **Regex host**: `^bigscreen-([^.]+)\.example\.com$` → `redirect.url` uses `$1`.
2. **Host fallback + path services**: default traffic redirects; paths matching `^/api/` proxy to a service.
3. **`${path.N}`**: path regex capture feeds `redirect.url`.
