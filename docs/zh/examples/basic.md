# 基础设置

此示例展示了一个简单的反向代理 Ingress 配置。

配置文件仓库路径：[examples/basic/](https://github.com/go-zoox/ingress/tree/master/examples/basic)。

## 最小配置

<<< @/../examples/basic/ingress.yaml

## 说明

- **port**：Ingress 监听端口 8080
- **rules**：定义路由规则
- **host**：匹配 `Host: example.com` 的请求
- **backend.service**：路由到端口 8080 上的 `backend-service`

## 测试

```bash
ingress run -c examples/basic/ingress.yaml
```

```bash
curl -H "Host: example.com" http://localhost:8080
```

## 多服务

<<< @/../examples/basic/multi-host.yaml

## 禁用规则

<<< @/../examples/basic/rule-disabled.yaml

当 `disabled.example.com` 的第一条规则 **`enabled: false`** 时，流量会落到下一条已启用规则（示例中为 handler 响应 `still live`）。
