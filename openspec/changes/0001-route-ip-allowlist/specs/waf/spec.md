# WAF — spec delta（0001-route-ip-allowlist）

> 本文件是本次变更的 **Requirement 增量**。实现完成、验收通过后，把以下 Requirement 合入
> `openspec/specs/waf/spec.md`，并归档 change 目录。

## ADDED Requirements

### Route-level IP allowlist

系统 SHALL 支持在单条路由上配置来源 IP 白名单，复用 WAF 的 IP 阶段：

- 配置位置：`rules[].waf.allow` / `rules[].waf.deny`（沿用 WAF 的 per-route patch 机制）；
- `allow` 非空 ⇒ 只有命中名单的来源放行，其余返回 403；
- `deny` 非空 ⇒ 命中即拒绝，优先于 `allow`；
- `allow`、`deny` 都为空 ⇒ 不限制（默认行为，SHALL 不改变现有配置）；
- 条目 SHALL 支持单个 IP 与 CIDR（IPv4 / IPv6），非法条目 SHALL 在加载 / 校验 / 保存阶段报错；
- 白名单 SHALL 在该路由 WAF 启用（`rules[].waf.enabled: true` 或全局 `waf.enabled: true`）时生效；
- 配置了 `allow`/`deny` 但该路由 WAF 未启用时，校验 SHALL 报错，避免安全配置静默失效。

### Trusted proxy client IP resolution

系统 SHALL 支持受信代理网段，用于安全地解析客户端 IP：

- 配置：`waf.trusted_proxies`（CIDR 列表，全局；per-route patch 可选）；
- 未配置时 SHALL 只使用 TCP `RemoteAddr`；
- 配置后，仅当 `RemoteAddr` 命中 `trusted_proxies` 时才采信 `X-Forwarded-For`；
- 采信时 SHALL 从 `X-Forwarded-For` 右侧向左取第一个不在 `trusted_proxies` 内的合法 IP 作为客户端 IP；若全部命中受信代理或没有合法 IP，则退回 `RemoteAddr`；
- `trusted_proxies` SHALL 拒绝 `0.0.0.0/0` 与 `::/0`（等于信任所有来源）；
- 存量 `trust_proxy` + `xff_index` 行为 SHALL 保持不变（未配置 `trusted_proxies` 时生效）。

### Admin console for route IP allowlist

管理后台 SHALL 提供路由级 IP 白名单的配置与展示：

- 路由编辑器 SHALL 提供「IP 白名单」区块：启用开关、允许列表、拒绝列表（高级）、是否同时启用攻击签名、客户端 IP 来源（只读继承）、拦截响应（403）；
- 保存 SHALL 只写入受管键（`enabled`、`disable_builtin`、`allow`、`deny`、`block_status_code`），SHALL NOT 覆盖 YAML 中其他 `waf` 配置（如 `rules`、`builtin_rules`、`allow_hosts`）；
- 关闭白名单后 SHALL 清理本功能写入的键，且当 `waf` 映射无其他配置时移除整个 `waf` 键；
- 路由列表 SHALL 展示白名单状态（条数 / 未限制）；路由详情 SHALL 展示白名单摘要；
- 全局 WAF 模块 SHALL 提供 `trusted_proxies` 编辑与「右起第一个非受信地址」算法说明。

### IP allowlist observability

系统 SHALL 复用现有可观测性，不新增独立事件表：

- 访问日志 SHALL 在 IP 白名单拦截时携带 `waf_block=1`；
- WAF 日志 SHALL 输出 `phase=ip allow` / `phase=ip deny` 与 `client_ip`；
- WAF 事件 SHALL 以 `rule=ip allow` / `rule=ip deny` 记录，供 Admin 事件列表筛选；
- 拦截响应 SHALL 使用 403 并复用 `error_pages` 的 403 页面。

## MODIFIED Requirements

### WAF client IP resolution

原「`trust_proxy` 布尔 + `xff_index`」的客户端 IP 判定保留，并新增 `trusted_proxies` 优先级：

- `trusted_proxies` 非空 → 受信代理算法（见上）；
- 否则 `trust_proxy: true` → `X-Forwarded-For[xff_index]`（兼容）；
- 否则 → `RemoteAddr`。
