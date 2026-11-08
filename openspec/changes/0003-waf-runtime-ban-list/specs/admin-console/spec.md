# admin-console — spec delta（0003-waf-runtime-ban-list）

> 本文件是本次变更的 **Requirement 增量**。实现完成、验收通过后，把以下 Requirement 合入
> `openspec/specs/admin-console/spec.md`，并归档 change 目录。

## ADDED Requirements

### Ban list page

管理后台 SHALL 提供独立的「封禁名单」页面：

- 页面 SHALL 分为三个 tab：生效中、已过期、操作审计；
- 「生效中」列表 SHALL 展示：类型、封禁目标、生效范围（全局 / 指定 Host）、到期时间与剩余时长、命中次数、操作人；
- 剩余时长短于阈值（≤2 小时）的行 SHALL 以警示色标出；
- 「已过期」列表 SHALL 保留过期记录并支持「重新封禁」一键回填；
- 「操作审计」列表 SHALL 展示 `waf.ban.*` 动作、详情与操作人；
- 列表 SHALL 支持按类型过滤与按 IP / UA / 原因 / 操作人关键字搜索。

### Ban from WAF event

WAF 事件列表 SHALL 提供从事件直接封禁的入口：

- 事件表每行 SHALL 提供「封禁 IP」与「封禁 UA」两个操作；
- 若事件缺少 User-Agent（空值），该行的「封禁 UA」操作 SHALL 不渲染；
- 点击 SHALL 打开预填抽屉，展示封禁目标、来源事件（事件 ID、命中的规则、Host）与影响面指标；
- 事件列表 SHALL 支持勾选多行后「批量封禁所选 IP」，批量操作 SHALL 只处理 IP，不提供批量封 UA。

### Ban confirmation with impact assessment

封禁确认抽屉 SHALL 在用户提交前展示影响面评估：

- SHALL 展示：该目标近 24h 命中次数、涉及的 Host 数、独立 UA 数、独立路径数；
- 当同一 IP 近 24h 出现大量不同 UA（默认阈值 20）时，SHALL 显示「疑似 NAT / 运营商出口，可能误伤真实用户」的警示；
- 当目标 IP 命中 `allow` 白名单时，SHALL 提示「运行时封禁优先于 allow 白名单，依然会生效」；
- 当目标 IP 命中 `trusted_proxies` 时，SHALL 显示阻断性错误并**禁用提交按钮**（除前端禁用外，后端同样拒绝）；
- 选择永久封禁时，SHALL 显示二次确认提示并将提交按钮文案改为「永久封禁（再次确认）」；
- UA 封禁 SHALL 明确标注「UA 是客户端可控字段，换 UA 即可绕过，仅作止血手段」。

### Ban scope and TTL controls

封禁抽屉 SHALL 提供作用域与时长控制：

- 作用域 SHALL 默认「全部 Host（全局）」，可切换为「指定 Host」并多选既有 Host；
- 时长 SHALL 提供预设：1 小时 / 24 小时 / 7 天 / 永久，默认 24 小时；
- 新增封禁 SHALL 提供原因备注输入。

### Ban input validation

封禁表单 SHALL 在客户端做即时校验：

- IP 类型：SHALL 校验 IPv4 点分十进制合法性（每段 0-255）或 CIDR（前缀长度 0-32），非法时 SHALL 内联报错并阻止提交；
- UA 正则类型：SHALL 校验正则可编译，非法时内联报错；
- 空值 SHALL 阻止提交。

### Ban list navigation

- 侧栏 SHALL 在「安全」分组下提供「封禁名单」入口；
- 侧栏徽标 SHALL 显示生效中的封禁条目数；
- 菜单权限 SHALL 复用 `waf` 相关权限码（`menu:waf-bans` 用于可见性，`waf:read` / `waf:write` 用于操作）。

## MODIFIED Requirements

### WAF page event actions

原 WAF 页事件表每行仅提供「详情」一个操作；本次扩展：

- 每行 SHALL 额外提供「封禁 IP」与「封禁 UA」；
- 既有「详情」操作与事件详情抽屉 SHALL 保持不变。

### WAF event action badges

原事件动作徽标仅有 `block` 与 `audit`；本次扩展：

- SHALL 新增 `ban` 徽标样式；
- 事件筛选的「动作」下拉 SHALL 新增 `ban` 选项。

## REMOVED Requirements

无。
