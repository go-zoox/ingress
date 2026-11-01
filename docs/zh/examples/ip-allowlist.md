# IP 白名单示例

源代码位于 [`examples/ip-allowlist/`](https://github.com/go-zoox/ingress/tree/master/examples/ip-allowlist)。

## 内网 + 指定 IP（按域名）

<<< @/../examples/ip-allowlist/ingress.yaml

## 位于 LB 后面（`trusted_proxies`）

<<< @/../examples/ip-allowlist/trusted-proxy.yaml

## 允许网段 + 单独拒绝一个 IP

<<< @/../examples/ip-allowlist/deny-and-allow.yaml

## 精确单个 IP

<<< @/../examples/ip-allowlist/single-ip.yaml

## 校验

```bash
ingress validate -c examples/ip-allowlist/ingress.yaml
```

语义详解见 [WAF 指南](../guide/waf.md#域名级-ip-白名单)。
