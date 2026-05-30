# 运行场景

源码位于 [`examples/scenarios/`](https://github.com/go-zoox/ingress/tree/master/examples/scenarios)。

## 默认 + overlay 列表（方案 C）

`active: default` 使用根配置；`live`、`drill` 为 overlay 场景。

<<< @/../examples/scenarios/design-option-c-list.yaml

## 可运行演示

<<< @/../examples/scenarios/ingress.yaml

## 电商日常 / 直播

单文件：基线为日常直连原站，`live` overlay 缓存商品读接口。

<<< @/../examples/scenarios/ecommerce.yaml

## 旧版独立文件

以下为引入 `scenarios` 前的独立配置，推荐改用 [`ecommerce.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/scenarios/ecommerce.yaml)。

<<< @/../examples/scenarios/ecommerce-daily.yaml

<<< @/../examples/scenarios/ecommerce-live-stream.yaml

## 校验

```bash
ingress validate -c examples/scenarios/design-option-c-list.yaml
ingress validate -c examples/scenarios/ingress.yaml
ingress validate -c examples/scenarios/ecommerce.yaml
```

语义与 Admin 用法见 [运行场景指南](../guide/scenarios.md)。
