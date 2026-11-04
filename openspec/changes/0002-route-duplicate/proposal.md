# change: 0002-route-duplicate（路由高级操作支持复制）

## 为什么

路由配置里「照着一条已有规则改一个域名/上游」是高频操作（灰度域名、复制生产规则到测试域名、同一上游换 Host 等）。现状只能：

1. 记住源规则的 backend / paths / WAF / 限流 / 缓存 / 安全 / 正向代理等全部字段，再逐项重填——字段多、易漏；
2. 或到「配置 → YAML」里手工复制一段 `rules[]` 再改 Host——容易缩进/字段写错，且要先理解 YAML 结构；
3. 复制出来的规则如果是同一 Host，列表顺序在前的会把它遮蔽，用户往往没意识到，最后得到一条「配了但永远不命中」的死规则。

产品原型（`prototypes/route-copy/`，已确认）验证了交互方向：**复制 = 在「高级」菜单里对当前规则做一次深拷贝，弹出预填的规则弹窗只让用户确认 Host**。

## 做什么

### 1. 两个复制入口

- 路由规则列表：行内「高级 ▾」菜单第一项「复制路由」；
- 路由详情页：`PageHeader.actions` 新增「复制路由」按钮，跳回 `/routes?copy=<ruleIndex>` 并自动打开同一个弹窗。

### 2. 复制弹窗（预填的规则编辑弹窗）

- 复用现有规则编辑分区表单（`RuleEntityFormSections`），标题为「复制路由」；
- 顶部固定源规则摘要：`源规则 #N · <host>`，以及「除 Host 外全部配置已复制」说明；
- 深拷贝源规则的全部配置：backend（service/handler/redirect + cache + auth + healthcheck + request/response 改写）、paths、connect_proxy、rate_limit、security、waf（IP 白名单与其他 WAF 键）；
- 副本与源规则互相独立，修改副本不影响源规则，反之亦然。

### 3. Host 默认值与冲突拦截

- 默认自动生成区分名：`api.example.com → api-copy.example.com`；
- 目标 Host 已存在时自动顺延：`api-copy.example.com` 已存在 → `api-copy-2.example.com`；
- 通配符保留 `*.` 前缀，只改 `*` 之后的第一个 label；`host_type` 显式为 `regex`（或 Host 含正则元字符）时**不自动改名**，只提示人工确认；
- Host 与任意既有规则重复时，弹窗内联报错（指明冲突规则编号与 Host），并禁用「创建副本」——避免创建永远被遮蔽的规则；
- Host 为空、含空白字符时同样拦截。

### 4. 副本落位与状态

- 启用状态跟随源规则（源禁用则副本禁用）；
- 插入位置默认「紧随源规则之后」（列表顺序即匹配优先级），可切换为「列表末尾」；
- 创建后：新行高亮约 3 秒，页面出现「未保存」状态（沿用现有 doc dirty），并显示可撤销提示条（约 8 秒）；
- 撤销：恢复到复制前的规则数组（只影响未保存草稿）。

### 5. 保存与发布

- 不新增后端接口、不改变 YAML 结构；复制只修改 `rules` 模块的文档草稿；
- 现有「保存与发布」链路（`SavePublishDrawer` + `buildPersistDiff` + `PublishModal`）正常展示新增规则的 `+` 行；
- `ingress validate` / reload 仍是最终兜底，重复 Host 若通过其它途径写入，也不会导致进程失败。

## 不做什么

- 不新增顶部 `rules` 之外的配置模块、不新增 API；
- 不做「复制到其它环境 / 其它实例」、批量复制、Path 级规则单独复制（只随整条规则复制）；
- 不做「复制模板 / 从示例创建」；
- 不改动编辑模式对重复 Host 的既有宽松行为（只在复制弹窗内强校验）；
- 不做复制历史的服务端审计记录（沿用现有配置保存/发布审计）。

## 影响面

| 层 | 文件（现状） | 变化 |
|---|---|---|
| admin web lib | `core/admin/web/src/lib/configEntities.ts` | 新增副本 Host 生成、Host 冲突校验、插入位置纯函数 |
| admin web 组件 | `components/config/RulesEditor.tsx` | 复制模式、弹窗横幅、插入位置、撤销提示、行高亮、`openCopy` handle |
| admin web 表单 | `components/config/RuleEntityFormSections.tsx`、`components/Form.tsx` | 支持 Host 字段级错误提示（可选 prop） |
| admin web 路由页 | `pages/RoutesPage.tsx`、`components/routes/RouteRulesManageTab.tsx` | 解析 `?copy=<ri>` 并自动打开复制弹窗 |
| admin web 详情页 | `pages/RouteDetailPage.tsx` | 新增「复制路由」入口 |
| 样式 | `styles/app.css` | 横幅 / 选项 / 撤销条 / 行高亮 / 字段错误 |
| 测试 | `src/lib/configEntities.copy.test.ts` | 纯函数与深拷贝语义单测 |
| 文档 | `AGENTS.md`、`openspec/` | 行为说明与本 change |
| 后端 / 配置 | — | 无变化（不需要，复制是客户端文档操作） |

## 验收

1. 列表行「高级 → 复制路由」打开弹窗，Host 预填 `-copy` 版本；源规则不变。
2. 源规则含 paths、IP 白名单、限流、缓存、正向代理、安全覆盖时，副本保存后这些字段逐项一致，YAML diff 只体现新增规则本身。
3. Host 改成已存在域名：Host 字段内联报错（含冲突规则编号），「创建副本」禁用。
4. 正则 Host 复制不自动改名并展示提示；用户改成新 Host 后可创建。
5. 默认插入源规则下一行；切「列表末尾」后插入末行；两种顺序保存后 YAML 顺序正确。
6. 源规则禁用 → 副本禁用；源启用 → 副本启用。
7. 创建后行高亮、「未保存」出现，撤销可恢复复制前的规则列表。
8. 详情页「复制路由」可打开同一弹窗，URL 中的 `copy` 参数打开后清理。
9. `pnpm test`、`pnpm build`（tsc -b + vite）通过；无新增后端接口。
10. 保存与发布：diff 显示新增规则，发布 reload 成功；复制正则/通配符规则时行为符合上面的命名规则。
