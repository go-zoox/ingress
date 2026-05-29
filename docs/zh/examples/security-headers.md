# 安全响应头示例

源代码：[`examples/security/`](https://github.com/go-zoox/ingress/tree/master/examples/security)。

## Profile：strict / api / embeddable

<<< @/../examples/security/profiles.yaml

验证：

```bash
ingress validate -c examples/security/profiles.yaml
```

详解见 [安全响应头指南](../guide/security-headers.md)。
