# Scenarios

Sources live under [`examples/scenarios/`](https://github.com/go-zoox/ingress/tree/master/examples/scenarios).

## Default + overlay list (方案 C)

`active: default` uses root config; `live` and `drill` apply overlays.

<<< @/../examples/scenarios/design-option-c-list.yaml

## Runnable demo

<<< @/../examples/scenarios/ingress.yaml

## E-commerce daily / live

Single file with baseline daily routing and a `live` overlay for product read caching.

<<< @/../examples/scenarios/ecommerce.yaml

## Wildcard baseline + exact host overlay

Baseline `*.example.com`; scenario `sh-live` inserts `sh.example.com` **before** the wildcard so Shanghai traffic uses overlay cache/upstream first.

<<< @/../examples/scenarios/wildcard-with-exact-overlay.yaml

## Legacy standalone files

These predate the unified `scenarios` block — use [`ecommerce.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/scenarios/ecommerce.yaml) for one-file switching.

<<< @/../examples/scenarios/ecommerce-daily.yaml

<<< @/../examples/scenarios/ecommerce-live-stream.yaml

## Validate

```bash
ingress validate -c examples/scenarios/design-option-c-list.yaml
ingress validate -c examples/scenarios/ingress.yaml
ingress validate -c examples/scenarios/ecommerce.yaml
ingress validate -c examples/scenarios/wildcard-with-exact-overlay.yaml
```

See [Scenarios guide](../guide/scenarios.md) for merge semantics, `default`, and Admin Console.
