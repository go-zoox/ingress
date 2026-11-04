# 路由复制（Route Duplicate）— 产品原型

单文件可点的产品原型，只覆盖**路由高级操作 → 复制路由**这一个新流程；其余页面沿用现有控制台形态（模拟数据，不连接 ingress）。

## 打开

```bash
# 方式一：直接双击，或浏览器打开
open prototypes/route-copy/index.html          # macOS
xdg-open prototypes/route-copy/index.html      # Linux

# 方式二：本地静态服务（推荐，交互稳定）
cd prototypes/route-copy && python3 -m http.server 8766
# http://127.0.0.1:8766
```

深链（方便评审直接看某个状态）：

- `index.html#copy=r2` — 打开 `api.example.com` 的复制弹窗（自动生成 `api-copy.example.com`）
- `index.html#copy=r3` — 演示「目标 Host 已被占用 → 自动顺延为 `admin-copy-2.example.com`」
- `index.html#copy=r6` — 演示「正则 Host 不自动改名 → 冲突拦截」
- `index.html#detail=r5` — 打开路由详情（第二个复制入口）

## 演示路径

1. **行内复制**：路由规则列表 → 某行「高级 ▾」→ 第一项「复制路由」→ 弹窗预填 Host（源 Host 加 `-copy`，冲突自动顺延）→ 改 Host / 类型 / 状态 / 插入位置 →「创建副本」。
2. **冲突拦截**：把 Host 改成已存在的域名（如 `www.example.com`），弹窗内立即红字提示「与规则 #1 重复」，创建按钮禁用。
3. **创建后**：副本插入源规则**下一行**并高亮，行内出现「副本 · 未发布」，顶部出现「未保存」，toast 可「撤销」。
4. **保存与发布**：点「保存与发布」→ diff 里可见新增规则（+ 行）→「保存并发布」走校验 → 写入 YAML → reload；发布成功后「未保存」与副本标记消失。
5. **详情入口**：点 Host 进入路由详情 → 右上角「复制路由」，同一套弹窗，源规则按详情页当前规则带入。
6. **边界**：`admin.example.com` 演示自动避让；`*.shop.example.com` 演示通配符只改 `*` 之后的部分；正则行演示「不自动改名 + 冲突拦截」。

## 设计要点（已实现于原型）

- **入口**：行内「高级」菜单第一项（列表批量场景）+ 路由详情页右上角（单条场景）。复制是非破坏性操作，不弹二次确认。
- **弹窗 = 预填的规则编辑弹窗**：顶部固定「源规则 #N · Host」摘要 + 已复制内容 chips（backend / paths / IP 白名单 / 限流 / 缓存 / 安全 / 正向代理），其余分区可切换查看，编辑聚焦在 Host。
- **Host 自动生成区分名**：源 Host 第一个 label 加 `-copy`；已存在则顺延 `-copy-2`；通配符保留 `*.` 只改后段；正则不改名、只提示人工确认。
- **冲突检测**：与任意既有规则 Host 相同即报错并禁用「创建副本」，错误信息指明冲突规则编号与 Host（避免创建出永远被遮蔽的死规则）。
- **深拷贝**：backend / paths / connect_proxy / rate_limit / security / waf 全部复制，副本与原规则互相独立。
- **插入位置**：默认紧随源规则之后（列表顺序 = 匹配优先级）；可切到列表末尾。
- **启用状态**：跟随源规则（已与产品确认）；真正的流量安全闸门是现有「保存 → 发布 reload」两段式。
- **可撤销**：创建后 toast 提供「撤销」，直接移除副本并恢复未保存状态。

## 已确认的产品决策

| 决策 | 选择 |
| --- | --- |
| 副本 Host 默认值 | 自动生成区分名（`api.example.com` → `api-copy.example.com`，占用则 `-copy-2`） |
| 副本启用状态 | 跟随源规则 |
| 插入位置 | 紧随源规则之后（可切换末尾） |

## 对真实实现的落点（仅位置说明，不含代码）

- `core/admin/web/src/components/config/RulesEditor.tsx` — 行操作 `menuItems` 增加「复制路由」，复制逻辑放到 `lib/configEntities.ts` 附近的纯函数（深拷贝 + Host 建议 + 冲突查询）。
- `ConfigEntityModal` / `RuleEntityFormSections` — 复用同一套分区表单，标题改为「复制路由」并加源规则副标题；Host 字段承载自动改名与冲突提示。
- 保存链路 — 仍走 `rules` 模块 doc 的保存 / 发布（`SavePublishDrawer` + `buildPersistDiff`），diff 里以「新增规则」呈现。
- 后端 — **无需新增接口**；`ingress validate` 仍是最终兜底（重复 Host 若允许存在，运行时也只是被前序规则遮蔽，不影响 reload 成功）。
- 路由详情页 `RouteDetailPage.tsx` — `PageHeader.actions` 加「复制路由」，复用同一个复制弹窗。

## 原型范围外

- 拖拽排序（原型保留了手柄，用「高级 → 上移 / 下移」替代）、Paths / 编辑 / 添加弹窗的完整表单。
- 复制到其它环境 / 批量复制 / Path 级复制、复制模板化。

## 验证

```bash
# 纯逻辑（Host 生成 / 冲突检测 / YAML 片段序列化）
node scripts/smoke.js

# 交互冒烟（需要本机 chromium；会自行拉起 headless 实例，校验 9222 空闲）
node scripts/ui-smoke.mjs
```

原型文件：`index.html`、`css/app.css`、`js/mock-data.js`、`js/copy-utils.js`、`js/app.js`。
