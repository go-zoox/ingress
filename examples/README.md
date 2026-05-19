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
| `path-routing/` | Path-based backends |
| `authentication/` | Basic and bearer auth |
| `ssl-tls/` | HTTPS, certs, global redirect |
| `advanced/` | Regex hosts, rewrites, health, **`service.mode`**, **`backend.cache`** (+ optional **`skip_vary`**), Redis |
| `redirect/` | Backend redirects and capture templates |
| `waf/` | IP lists, custom signatures, `rules[].waf` overlays |

Compose production configs by merging patterns from these files; there is no longer a single monolithic sample in-repo.

**HTTP response cache** (`backend.cache`): `examples/advanced/http-response-cache.yaml` (service/handler/redirect; live **`skip_vary`** demo against **`https://httpbin.zcorky.com`** on host `api-cached.httpbin.work`). **`cache` engine + Redis**: `examples/advanced/redis-cache.yaml`.

The documentation site (`docs/examples/`) embeds these files via VitePress code snippets so examples stay in one place.
