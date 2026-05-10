# TODO List

本文档列出了 Ingress 项目计划实现的功能和改进。这些功能基于对 Nginx 和 Kubernetes Ingress 标准功能的调研，旨在使 Ingress 成为一个功能完整、生产就绪的反向代理解决方案。

## 优先级说明

- **P0 (高优先级)**: 核心功能，反向代理的基本能力
- **P1 (中优先级)**: 重要功能，提升生产环境可用性
- **P2 (低优先级)**: 增强功能，扩展场景支持

---

## P0 - 高优先级功能

### 1. 负载均衡 (Load Balancing)

**状态**: 🔴 未实现  
**描述**: 当前仅支持单个后端服务，需要支持多后端服务器池和多种负载均衡算法。

**功能需求**:
- [ ] 支持多个后端服务器配置（upstream pool）
- [ ] 实现负载均衡算法：
  - [ ] Round-robin（轮询）
  - [ ] Least connections（最少连接）
  - [ ] IP hash（IP 哈希）
  - [ ] Weighted round-robin（加权轮询）
- [ ] 健康检查与自动故障转移
- [ ] 会话保持（sticky session/cookie-based affinity）
- [ ] 后端服务器权重配置

**参考**: Nginx upstream, Kubernetes Service endpoints

---

### 2. 限流/速率限制 (Rate Limiting)

**状态**: 🟡 部分实现（有依赖但未使用）  
**描述**: 已有 `ratelimit` 依赖，但未集成到核心功能中。

**功能需求**:
- [ ] 基于 IP 的限流
- [ ] 基于用户/认证的限流
- [ ] 基于路径的限流
- [ ] 实现令牌桶/漏桶算法
- [ ] 突发流量控制
- [ ] 自定义限流响应（429 Too Many Requests）
- [ ] 限流白名单配置

**参考**: Nginx limit_req, API Gateway rate limiting

---

### 3. 访问控制 (Access Control)

**状态**: 🔴 未实现  
**描述**: 基础的访问控制能力，包括 IP 过滤和 CORS 支持。

**功能需求**:
- [ ] IP 白名单/黑名单
- [ ] CIDR 网段支持
- [ ] 地理位置限制（GeoIP）
- [ ] CORS（跨域资源共享）配置
- [ ] 请求大小限制（body size limit）
- [ ] 请求头大小限制
- [ ] 请求方法限制（GET, POST, etc.）

**参考**: Nginx allow/deny, CORS middleware

---

### 4. 服务治理 (Service Governance)

**状态**: 🔴 未实现  
**描述**: 提升服务可靠性的核心能力。

**功能需求**:
- [ ] 熔断器（Circuit Breaker）
  - [ ] 失败率阈值
  - [ ] 半开状态管理
  - [ ] 自动恢复机制
- [ ] 降级策略（Fallback response）
- [ ] 重试机制
  - [ ] 指数退避算法
  - [ ] 最大重试次数配置
  - [ ] 可重试状态码配置
- [ ] 超时配置优化
- [ ] 连接池管理

**参考**: Hystrix, Resilience4j, Envoy retry policy

---

### 5. 流量管理 (Traffic Management)

**状态**: 🔴 未实现  
**描述**: 支持灰度发布、A/B 测试等高级流量控制。

**功能需求**:
- [ ] 灰度发布/金丝雀部署（Canary Deployment）
  - [ ] 按百分比流量分配
  - [ ] 按权重流量分配
  - [ ] 基于 Header/Cookie 的流量路由
- [ ] A/B 测试支持
- [ ] 流量镜像（Shadow/Mirror Traffic）
- [ ] 流量分割（Traffic Splitting）
- [ ] 蓝绿部署支持

**参考**: Kubernetes Ingress canary, Istio VirtualService

---

### 6. 配置热更新优化 (Hot Reload)

**状态**: 🟡 部分实现  
**描述**: 当前有 reload 功能，但需要优化为零停机配置重载。

**功能需求**:
- [ ] 零停机配置重载
- [ ] 配置验证（validation before apply）
- [ ] 配置回滚机制
- [ ] 动态配置 API（REST API）
- [ ] 配置变更通知
- [ ] 配置版本管理

**参考**: Traefik dynamic configuration, Envoy hot restart

---

## P1 - 中优先级功能

### 7. 压缩支持 (Compression)

**状态**: 🟡 部分实现（有依赖但未使用）  
**描述**: 已有 `gzip` 依赖，需要集成到响应处理中。

**功能需求**:
- [ ] Gzip 压缩
- [ ] Brotli 压缩
- [ ] 可配置压缩级别
- [ ] 条件压缩（基于 Content-Type）
- [ ] 最小压缩大小配置
- [ ] Vary header 自动设置

**参考**: Nginx gzip, HTTP compression

---

### 8. WebSocket 支持

**状态**: 🟡 部分实现（有依赖但未使用）  
**描述**: 已有 `websocket` 依赖，需要实现 WebSocket 代理功能。

**功能需求**:
- [ ] WebSocket 代理
- [ ] WebSocket 升级处理
- [ ] 长连接管理
- [ ] WebSocket 健康检查
- [ ] WebSocket 超时配置

**参考**: Nginx proxy_http_version 1.1, WebSocket proxy

---

### 9. gRPC 支持

**状态**: 🟡 部分实现（有依赖但未使用）  
**描述**: 已有 `grpc` 依赖，需要实现 gRPC 代理功能。

**功能需求**:
- [ ] gRPC 代理
- [ ] HTTP/2 支持
- [ ] gRPC 健康检查
- [ ] gRPC 超时配置
- [ ] gRPC 负载均衡

**参考**: Envoy gRPC, Traefik gRPC support

---

### 10. 可观测性完善 (Observability)

**状态**: 🟡 部分实现  
**描述**: 已有 Prometheus/OpenTelemetry 依赖，需要完善集成。

**功能需求**:
- [ ] 结构化日志（JSON 格式）
- [ ] 日志级别配置
- [ ] Metrics 暴露（Prometheus endpoint）
- [ ] 分布式追踪（OpenTelemetry）
- [ ] 请求日志（Access Log）配置
- [ ] 性能指标收集：
  - [ ] 请求延迟（P50, P95, P99）
  - [ ] 吞吐量（QPS）
  - [ ] 错误率
  - [ ] 连接数统计
- [ ] 告警集成

**参考**: Prometheus metrics, OpenTelemetry, Nginx access log

---

### 11. WAF 基础功能 (Web Application Firewall)

**状态**: 🟡 部分实现（配置中有字段但未实现）  
**描述**: 配置中有 `waf` 字段，但功能未实现。

**功能需求**:
- [ ] SQL 注入防护
- [ ] XSS（跨站脚本）防护
- [ ] CSRF 防护
- [ ] 请求频率限制
- [ ] 恶意请求检测
- [ ] 自定义规则引擎
- [ ] WAF 白名单/黑名单

**参考**: ModSecurity, Cloudflare WAF

---

### 12. 服务发现 (Service Discovery)

**状态**: 🔴 未实现  
**描述**: 当前仅支持静态配置，需要支持动态服务发现。

**功能需求**:
- [ ] DNS 服务发现
- [ ] Kubernetes Service 发现
- [ ] Consul 集成
- [ ] Eureka 集成
- [ ] Nacos 集成
- [ ] 动态后端更新
- [ ] 服务注册/注销通知

**参考**: Kubernetes Endpoints, Consul Connect, Service Mesh

---

### 13. 连接管理 (Connection Management)

**状态**: 🔴 未实现  
**描述**: 连接池和连接复用优化。

**功能需求**:
- [ ] 连接池配置（最大连接数）
- [ ] 空闲连接超时
- [ ] Keep-Alive 配置
- [ ] 连接复用
- [ ] 慢连接检测
- [ ] 连接数限制

**参考**: Nginx keepalive, HTTP connection pooling

---

### 14. 高级认证完善 (Advanced Authentication)

**状态**: 🟡 部分实现  
**描述**: JWT/OAuth2/OIDC 在配置中有定义，但可能未完全实现。

**功能需求**:
- [ ] JWT 验证完整实现
  - [ ] JWT 签名验证
  - [ ] JWT 过期检查
  - [ ] JWT 声明（claims）验证
- [ ] OAuth2 完整流程实现
- [ ] OIDC 完整流程实现
- [ ] mTLS（双向 TLS）支持
- [ ] API Key 认证
- [ ] 认证缓存

**参考**: OAuth2, OIDC, JWT standards

---

## P2 - 低优先级功能

### 15. 协议转换 (Protocol Conversion)

**状态**: 🔴 未实现  
**描述**: 支持不同协议之间的转换。

**功能需求**:
- [ ] HTTP 到 gRPC 转换
- [ ] HTTP 到 Dubbo 转换
- [ ] 协议升级/降级
- [ ] 协议适配器

**参考**: Protocol adapters, API Gateway protocol conversion

---

### 16. 请求/响应体修改 (Body Modification)

**状态**: 🔴 未实现  
**描述**: 当前仅支持 header 修改，需要支持 body 修改。

**功能需求**:
- [ ] 请求体修改
- [ ] 响应体修改
- [ ] 内容替换/过滤
- [ ] JSON 路径修改
- [ ] XML 处理
- [ ] 流式处理支持

**参考**: Nginx sub_filter, Response transformation

---

### 17. 其他增强功能

**功能需求**:
- [ ] 请求缓冲配置
- [ ] 响应缓冲配置
- [ ] 文件上传大小限制
- [ ] 慢请求日志
- [ ] 请求 ID 生成和追踪
- [ ] 自定义错误页面
- [ ] 请求/响应采样
- [ ] 多租户支持

---

## 实现进度跟踪

### 总体进度

- **P0 功能**: 0/6 (0%)
- **P1 功能**: 0/8 (0%)
- **P2 功能**: 0/3 (0%)
- **总计**: 0/17 (0%)

### 最近更新

- 2024-XX-XX: 创建 TODO List，基于 Nginx 和 Kubernetes Ingress 标准功能调研

---

## 贡献指南

欢迎贡献代码实现这些功能！在开始实现之前，请：

1. 查看相关的 Issue 和 PR
2. 参阅 [Ingress on GitHub](https://github.com/go-zoox/ingress)
3. 在 Issue 中讨论实现方案
4. 提交 PR 时参考现有的代码风格

---

## 参考资源

- [Nginx Documentation](https://nginx.org/en/docs/)
- [Kubernetes Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
- [Envoy Proxy](https://www.envoyproxy.io/)
- [Traefik Documentation](https://doc.traefik.io/traefik/)
- [API Gateway Patterns](https://microservices.io/patterns/apigateway.html)
