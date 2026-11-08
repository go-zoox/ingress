# WAF 运行时封禁名单 — 产品原型

单机场景下的**可点击原型**，演示「后台从拦截记录直接封禁 IP / UA」的完整交互。全部为模拟数据，不连接真实 ingress 进程，不修改任何配置。

## 打开方式

```bash
# 在仓库根目录
cd prototypes/waf-ban
python3 -m http.server 8766
# 浏览器访问 http://127.0.0.1:8766
```

也可以直接双击 `index.html`。

## 原型范围

| 页面 | 说明 |
|------|------|
| WAF 事件 | 事件表每行新增「封禁 IP」「封禁 UA」入口；支持勾选批量封禁 |
| 封禁名单 | 生效中 / 已过期 / 操作审计 三个 tab；详情、续期、解封 |

4 个抽屉：

1. **从事件封禁** — 展示影响面评估（近 24h 命中、独立 Host/UA/路径）、误封风险提示、生效范围、时长、原因
2. **新增封禁** — IP/CIDR 与 User-Agent 两种类型，实时校验 + 危险目标拦截
3. **封禁详情** — 命中统计、来源、可续期 / 立即解封
4. **批量封禁** — 勾选多个事件后批量封 IP

## 关键产品决策（原型里已体现）

- **运行时名单**：封禁立即生效，不触发 reload、不清空响应缓存、不影响限流计数器
- **独立于 `waf.enabled`**：WAF 总开关关闭时，封禁依然生效
- **默认全局，可按 Host 限定**：多租户场景可限定范围避免误伤
- **封禁优先于 allow 白名单**：与现有 `IPAllowlistFormFields` 的「临时封禁」语义一致
- **危险目标拦截**：目标命中 `trusted_proxies` 时**拒绝封禁**并禁用提交按钮（封它 = 封全站）
- **NAT 误伤预警**：同一 IP 出现大量不同 UA 时提示可能误伤真实用户
- **UA 定位为止血**：明确标注「客户端可控，换 UA 即可绕过，非防御手段」

## 数据结构（供实现参考）

封禁条目：`kind`(ip/ip_cidr/ua_contains/ua_regex)、`value`、`hosts[]`、`reason`、`source`、`actor`、`hits`、`expires_at`、`enabled`、`created_at`。

审计动作：`waf.ban.add` / `waf.ban.remove` / `waf.ban.renew` / `waf.ban.expire`。

建议 API：

```
GET    /api/v1/waf/bans              列表
POST   /api/v1/waf/bans              新增
PATCH  /api/v1/waf/bans/:id          改 TTL / 启停
DELETE /api/v1/waf/bans/:id          解封
POST   /api/v1/waf/bans/preview      封禁前试算影响面
```

## 与现有后台的关系

视觉体系、CSS 变量、`.panel` / `table.data` / `.drawer` / `.filter-grid` 等类名与
`core/admin/web/src/styles/app.css` 保持一致，落地时可直接复用现有组件与样式。
