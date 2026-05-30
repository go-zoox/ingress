# Example configurations

Runnable YAML samples live under this directory. Validate any file before deploy:

```bash
ingress validate -c examples/basic/ingress.yaml
```

Each `rules[].backend` and `paths[].backend` picks a mode from **`backend.service`**, **`backend.handler`**, or **`backend.redirect`**. **`backend.type` is optional**: Ingress **infers** `service`, `handler`, or `redirect` at load/validate when exactly one of those blocks clearly applies. Add **`type` explicitly** only if validation reports an **ambiguous** backend (multiple blocks look populated). Details: [`docs/guide/routing.md`](docs/guide/routing.md).

**`backend.service.mode`** (`internal` default, or `external` for upstream `Host` aligned to `service.name`) applies on **proxy** backends; legacy **`backend.mode`** is still accepted if it matches. See [`docs/guide/rewriting.md`](../docs/guide/rewriting.md) and **`examples/advanced/service-mode-external-mixed.yaml`**.

Several files **mix explicit `backend.type` and omission on purpose** (for example `examples/basic/ingress.yaml`, `examples/ssl-tls/route-redirect.yaml`, `examples/redirect/capture-and-mixed.yaml`) so you can compare equivalent spellings side by side in one runnable config.

| Directory | Topic |
|-----------|--------|
| `basic/` | Minimal host routing |
| `path-routing/` | Path-based backends, **`strip_prefix: true`** |
| `authentication/` | Basic and bearer auth |
| `ssl-tls/` | HTTPS, certs, global redirect |
| `advanced/` | Regex hosts, rewrites, health, **`service.mode`**, **`backend.cache`** (+ optional **`skip_vary`**), Redis |
| `redirect/` | Backend redirects and capture templates |
| `handler/` | **`backend.handler`** — `static_response`, `file_server`, `templates`, `script` |
| `waf/` | IP lists, custom signatures, `rules[].waf` overlays |
| `admin-console/` | **Admin UI** demo routes, WAF, sample `access.log` (`admin.enabled: true` in ingress.yaml) |
| `admin-auth/` | **Admin Console login** — default `none` (`open-no-auth.yaml`) + opt-in `basic` (`ingress.yaml`) |

Compose production configs by merging patterns from these files; there is no longer a single monolithic sample in-repo.

**HTTP response cache** (`backend.cache`): `examples/advanced/http-response-cache.yaml` (service/handler/redirect; live **`skip_vary`** demo against **`https://httpbin.zcorky.com`** on host `api-cached.httpbin.work`). **Per-path rules**: `examples/advanced/http-response-cache-paths.yaml`. **POST + JSON `key_json`**: `examples/advanced/http-response-cache-post-json.yaml`. **`cache` engine + Redis**: `examples/advanced/redis-cache.yaml`.

The documentation site (`docs/examples/`) embeds these files via VitePress code snippets so examples stay in one place.
