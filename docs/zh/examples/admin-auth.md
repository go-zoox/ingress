# Admin 认证与 RBAC

最小 Admin Console **本地 Basic 登录**示例，含内置 RBAC 角色。

源码：[`examples/admin-auth/`](https://github.com/go-zoox/ingress/tree/master/examples/admin-auth)。

## Basic 登录（默认）

<<< @/../examples/admin-auth/ingress.yaml{yaml}

| 项 | 值 |
|----|-----|
| Admin 地址 | `http://127.0.0.1:9080` |
| 默认账号 | `admin` / `admin`（来自 `admin.auth.basic`） |
| RBAC 数据库 | 与本 YAML 同目录的 `./admin-auth.db` |

## 校验与运行

```bash
ingress validate -c examples/admin-auth/ingress.yaml
ingress run -c examples/admin-auth/ingress.yaml
```

登录后在侧栏 **权限** 中管理用户、角色与权限。

## 开放模式（仅开发）

<<< @/../examples/admin-auth/open-no-auth.yaml{yaml}

`admin.auth.type: none` 跳过登录页，仅适合 localhost 或可信网络。

## 相关文档

- [Admin 控制台指南 · 认证与 RBAC](/zh/guide/admin#认证与-rbac)
- 含示例日志与 WAF 的完整演示包：[Admin 控制台示例](/zh/examples/admin-console)
