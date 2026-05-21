# Ingress 运维管理后台 — 产品原型

单机部署场景下的**可点击原型**（全部为模拟数据，不连接真实 ingress 进程）。

## 打开方式

```bash
# 在仓库根目录
cd prototypes/admin-console
python3 -m http.server 8765
# 浏览器访问 http://127.0.0.1:8765
```

或直接双击 `index.html`（部分交互在 `file://` 下可能受限，推荐用上面的本地服务）。

## 原型范围

| 页面 | 说明 |
|------|------|
| 总览 | 进程状态、规则数、WAF/TLS 摘要、最近事件 |
| 路由 | 编译后路由表、试匹配（host + path） |
| WAF | 全局/规则、近期 block/audit |
| TLS | 证书列表与过期提醒 |
| 配置 | YAML 编辑、校验、diff、保存落盘、发布 reload |
| 日志 | 简易访问日志查询（过滤/分页） |

## 后续实现对照

原型中的按钮与 API 命名仅作产品约定，实现时可对齐：

- `POST /v1/config/validate`
- `POST /v1/reload`
- `GET /v1/routes`、`POST /v1/routes/match`
- `GET /v1/logs?...`

修改原型 UI/流程时直接改本目录内静态文件即可，无需编译。
