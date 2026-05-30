# 维护模式示例

源码位于 [`examples/maintenance/`](https://github.com/go-zoox/ingress/tree/master/examples/maintenance)。

## 全局 + 路由级维护

<<< @/../examples/maintenance/ingress.yaml

## 校验

```bash
ingress validate -c examples/maintenance/ingress.yaml
```

详解见 [维护模式指南](../guide/maintenance.md)。
