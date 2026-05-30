# Runtime scenarios

Ingress can define **named runtime overlays** in a single `ingress.yaml`. The root file is the **baseline** (daily traffic). Switch `scenarios.active` to merge an overlay for live streaming, drills, or other modes without maintaining separate config files.

## Schema (方案 C)

```yaml
scenarios:
  active: default   # or live, drill, …
  items:
    - id: live
      label: 直播
      description: Product read caching for high concurrency
      overlay:
        cache:
          host: redis.internal
          prefix: "ingress:live:"
        rules:
          - host: shop.example.com
            backend:
              cache:
                enabled: true
                default: bypass
                paths:
                  - match: /api/v1/products
                    match_type: prefix
                    action: cache
                    ttl: 60
```

| Field | Description |
|-------|-------------|
| `scenarios.active` | Current scene id. **`default`** (or empty) = root config, **no overlay**. |
| `scenarios.items[]` | Overlay scenes only (`id`, `label`, `description`, `overlay`). |
| `items[].overlay` | Partial config merged onto the baseline at prepare/reload time. |

### Reserved `default` scene

- **`scenarios.active: default`** uses the root `ingress.yaml` as-is.
- Do **not** add an `items[]` entry with `id: default` — it is reserved.
- Admin Console always lists **默认** as the first virtual scene.

### Overlay merge rules

Supported overlay keys: `cache`, `rate_limit`, `waf`, `maintenance`, `security`, `rules`.

- Top-level keys **deep-merge** onto the baseline.
- `rules[]` patches match by **`host`** and merge into existing route rows (unknown hosts fail validation).

### Runtime override

Set **`INGRESS_SCENARIO=<id>`** to override `scenarios.active` without editing YAML (useful in containers).

```bash
INGRESS_SCENARIO=live ingress run -c ingress.yaml
```

## Validate and switch

```bash
ingress validate -c examples/scenarios/ingress.yaml
ingress run -c examples/scenarios/ingress.yaml
```

Switch on disk + reload:

- Edit `scenarios.active` and `SIGHUP` / Admin **Publish**
- **`PUT /api/v1/scenarios/active`** with `{ "id": "live" }` (Admin API)

## Admin Console

When **`admin.enabled: true`**, open **维护 → 场景管理**:

- Edit overlay scenes (CRUD), pick **当前场景**, **保存与发布**
- **切换生效** updates `scenarios.active` and reloads without leaving the page

Config module **场景** under **配置** edits the same YAML block.

## Examples

Runnable samples: [`examples/scenarios/`](https://github.com/go-zoox/ingress/tree/master/examples/scenarios) — see [Scenarios examples](../examples/scenarios.md).

Design notes: [`docs/plans/2026-05-30-scenarios-config-design.md`](https://github.com/go-zoox/ingress/blob/master/docs/plans/2026-05-30-scenarios-config-design.md).
