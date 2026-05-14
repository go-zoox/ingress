---
layout: home

hero:
  name: Ingress
  text: 反向代理
  tagline: 一个简单、强大、灵活的反向代理
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 查看 GitHub
      link: https://github.com/go-zoox/ingress

features:
  - icon: 🚀
    title: 易于使用
    details: 使用 YAML 文件进行简单配置。几分钟内即可开始，设置最少。
  - icon: 🔒
    title: 安全
    details: 内置认证支持（Basic、Bearer、JWT、OAuth2、OIDC）和 SSL/TLS 终止。
  - icon: ⚡
    title: 高性能
    details: 高效路由，支持缓存（内存或 Redis），以获得最佳性能。
  - icon: 🎯
    title: 灵活路由
    details: 支持精确、正则表达式和通配符主机匹配，以及基于路径的路由。
  - icon: 🏥
    title: 健康检查
    details: 内置健康检查支持，用于外部和内部服务监控。
  - icon: 🔄
    title: 请求重写
    details: 路径、请求头、查询参数重写；backend.mode（internal / external）控制反代外网源时上游 Host 的默认行为。

---

## 快速开始

安装 Ingress：

```bash
go install github.com/go-zoox/ingress@latest
```

启动服务器：

```bash
# 使用默认配置启动（端口 8080）
ingress run

# 使用自定义配置文件启动
ingress run -c ingress.yaml
```

基本配置示例：

```yaml
version: v1
port: 8080

rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

不写 **`backend.type`** 与写 **`type: service`** 可等价——并排示例见 **`examples/basic/ingress.yaml`**。

`backend.type` 可选：当恰好配置 **`service`**、**`handler`**、**`redirect`** 中的一种时，Ingress 会自动推断类型。

更多详情，请参阅[快速开始指南](/zh/guide/getting-started)。
