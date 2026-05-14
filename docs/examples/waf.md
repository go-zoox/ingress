# WAF examples

Sources live under [`examples/waf/`](https://github.com/go-zoox/ingress/tree/master/examples/waf).

## Audit-only (do not block)

<<< @/../examples/waf/log-only-audit.yaml

## IP deny list + custom path rule

<<< @/../examples/waf/deny-and-custom.yaml

## Route-level `rules[].waf` overrides by rule id

<<< @/../examples/waf/rule-merge-by-id.yaml

## Validate

```bash
ingress validate -c examples/waf/log-only-audit.yaml
```

See [WAF guide](../guide/waf.md) for semantics.
