# WAF 示例

源代码位于 [`examples/waf/`](https://github.com/go-zoox/ingress/tree/master/examples/waf)。

## 仅审计（不拦截）

<<< @/../examples/waf/log-only-audit.yaml

## IP 拒绝 + 自定义路径规则

<<< @/../examples/waf/deny-and-custom.yaml

## 路由级 `rules[].waf` 按规则 id 覆盖

<<< @/../examples/waf/rule-merge-by-id.yaml

## 校验

```bash
ingress validate -c examples/waf/log-only-audit.yaml
```

详解见 [WAF 指南](../guide/waf.md)。
