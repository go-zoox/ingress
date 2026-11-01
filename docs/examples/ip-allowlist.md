# IP allowlist examples

Sources live under [`examples/ip-allowlist/`](https://github.com/go-zoox/ingress/tree/master/examples/ip-allowlist).

## Internal + specific IPs per host

<<< @/../examples/ip-allowlist/ingress.yaml

## Behind a load balancer (`trusted_proxies`)

<<< @/../examples/ip-allowlist/trusted-proxy.yaml

## Allow with a denied exception

<<< @/../examples/ip-allowlist/deny-and-allow.yaml

## Exact single IP

<<< @/../examples/ip-allowlist/single-ip.yaml

## Validate

```bash
ingress validate -c examples/ip-allowlist/ingress.yaml
```

See [WAF guide](../guide/waf.md#route-level-ip-allowlist-per-host) for semantics.
