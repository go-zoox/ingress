# Ingress 运维管理后台

单机运维控制台：**React + TypeScript + Vite + pnpm**（`web/`），**go-zoox + gormx (SQLite)**（`console/`）。

## CLI

```bash
ingress admin -c examples/admin-console/admin.yaml
```

默认 `-c` 即上述路径（从仓库根目录执行）。

## 目录

```
admin/
├── web/          # 前端
├── console/      # 后端 API + embed 静态资源
└── admin.yaml.example  # 指向 examples/admin-console/admin.yaml
```

**样例配置与日志文件**在 [`../examples/admin-console/`](../examples/admin-console/)（首次启动空 DB 会写入示例 WAF/审计行；日志页与概览指标只读配置的 `access.log` / `error.log`）：

| 文件 | 说明 |
|------|------|
| `admin.yaml` | Admin 服务配置（`log_path`、`error_log_path` 等同目录样例日志） |
| `ingress.yaml` | 原型对齐的 ingress 路由 / WAF / 健康检查样例 |
| `access.log` / `error.log` | 约 90 天样例日志（可用 `scripts/gen_sample_data.py` 重新生成） |

## 快速开始

```bash
# 仓库根目录
ingress validate -c examples/admin-console/ingress.yaml
ingress admin -c examples/admin-console/admin.yaml

# 前端开发（另一终端）
cd admin/web && pnpm install && pnpm dev
# http://127.0.0.1:5173 ，/api 代理到 9080
```

`examples/admin-console/admin.yaml` 里 `web.dev_proxy: true` 时只起 API；生产构建后由 Go embed 托管 UI：

```bash
cd admin && make build
ingress admin -c examples/admin-console/admin.yaml
```

## API

基路径 `/api/v1`（无鉴权，建议仅本机或内网）。路径相对 **admin.yaml 所在目录**解析，与 shell 当前目录无关。
