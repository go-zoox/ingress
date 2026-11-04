# Tasks — 0002-route-duplicate

> 研发计划：阶段 0–1 已完成（原型确认 + openspec）。阶段 2–6 按顺序实现，7 为发布门禁。
> 状态：`[ ]` 待办 / `[x]` 完成；完成后回填验证方式。

## 研发计划（阶段 / 产出 / 依赖）

| 阶段 | 内容 | 产出 | 依赖 |
|---|---|---|---|
| 0 | 产品原型与交互定版 | `prototypes/route-copy/`（已确认：自动区分名 / 启用跟随源 / 紧随源规则之后） | — |
| 1 | openspec（proposal / design / task / spec delta） | 本目录 | 阶段 0 |
| 2 | lib 纯函数 + 单测 | `configEntities.ts` 新增 helper、`configEntities.copy.test.ts` | 阶段 1 |
| 3 | RulesEditor 复制模式 | 弹窗、横幅、Host 校验、插入位置、撤销、行高亮、`openCopy` handle | 阶段 2 |
| 4 | 入口串联 | 行内菜单、「复制路由」按钮、`?copy=` 深链 | 阶段 3 |
| 5 | 样式 / 可访问性 | `app.css` 新增类、字段错误 `aria-invalid`、错误文案 | 阶段 3–4 |
| 6 | 文档 | `AGENTS.md` 补一节 | 阶段 3–5 |
| 7 | 发布门禁 | `pnpm test` / `pnpm build`；版本 `1.23.1`；tag `v1.23.1`；推送 | 阶段 2–6 |

写作用域（避免并发冲突）：阶段 2 只动 `core/admin/web/src/lib/`；阶段 3 动 `components/config/`；阶段 4 动 `pages/`；阶段 5 动 `styles/app.css`；阶段 6 动 `AGENTS.md`。四者可顺序执行，无跨文件写冲突。

## 0. 确认

- [x] 产品原型确认（`prototypes/route-copy/`）
- [x] 本 change（proposal + design + spec delta）确认

## 1. 规格

- [x] `openspec/changes/0002-route-duplicate/`：proposal / design / tasks / `specs/admin-console/spec.md`
- [x] Spec delta 覆盖：两入口、深拷贝、Host 生成与冲突拦截、插入位置、启用跟随、撤销、无后端变更

## 2. lib 纯函数与单测

- [x] `configEntities.ts`：`effectiveHostTypeForHost`（对齐 `core/compile.go` 推断顺序）
- [x] `configEntities.ts`：`suggestCopyHost(host, existingHosts, hostType?)`——`-copy` / 占用顺延 / wildcard 保留 `*.` / regex 不改名 / 副本的副本
- [x] `configEntities.ts`：`findHostConflict` + `copyHostValidation`（空、空白、重复，大小写不敏感）
- [x] `configEntities.ts`：`insertCopiedRule(rules, sourceIndex, 'after' | 'end', rule)`，返回新数组与插入下标
- [x] 单测 `configEntities.copy.test.ts`：Host 生成矩阵、冲突检测、插入位置、深拷贝保真（`waf.rules` / backend 未知键 / Path 级配置保留）
- [x] 定向运行：`pnpm exec vitest run src/lib/configEntities.copy.test.ts`

## 3. RulesEditor 复制模式

- [x] `EntityRowActions.menuItems` 第一项「复制路由」→ `openCopy(i)`（`Paths` / `详情` 顺延）
- [x] `RulesEditorHandle` 增加 `openCopy(index)`（`useImperativeHandle`）
- [x] 复制状态：`copySource: { index, host, suggestion } | null`、`copyPosition`、`highlightIndex`、撤销提示
- [x] `openCopy`：`ruleToForm(源)` 预填 + `suggestCopyHost` 覆盖 Host + 默认 `position: after` + 兼容 `activeSection='basic'`
- [x] `save()`：复制分支深拷贝源规则 → `formToRule(draft, clone)` → `insertCopiedRule`；编辑/新增分支保持不变
- [x] 复制横幅：源规则摘要、复制内容说明、插入位置 select、自动改名 / 冲突顺延 / 正则提示
- [x] Host 实时校验：错误阻塞保存；`disableSave = ruleSaveDisabled(draft) || copyHostError`
- [x] 创建后：行高亮（约 3.2s）+ 底部撤销提示条（约 8s，快照回滚）+ 清理计时器
- [x] 副本 `enabled` 跟随源规则（来自 `ruleToForm`，允许用户改回）

## 4. 入口串联

- [x] `RoutesPage`：解析 `?copy=<ruleIndex>`，传给 `RouteRulesManageTab`
- [x] `RouteRulesManageTab`：doc 加载完成后调用 `openCopy`，并通知父级用 `replace` 清除 `copy` 参数（防重复弹窗）
- [x] `RouteDetailPage`：`PageHeader.actions` 新增「复制路由」（Copy 图标）→ `/routes?copy=<rule_index>`
- [x] 深链在 rules 模块未加载完成 / 下标越界时安全降级（不崩溃、不弹空弹窗）

## 5. 样式与可访问性

- [x] `app.css`：`.copy-route-banner` / `.copy-route-options` / `.copy-undo-bar` / `.row-flash` / `.form-error-text`
- [x] `FormField` 增加可选 `error`：字段级红字 + `aria-invalid`，不影响既有调用方
- [x] `RuleEntityFormSections` 透传 `hostError`（仅 rule 变体基础分区）
- [x] 键盘：Esc 关闭沿用 `Drawer` 既有行为；错误文案对屏幕阅读器可见

## 6. 文档

- [x] `AGENTS.md`：Admin console 小节补充「路由复制」行为、入口、纯函数位置与校验语义
- [x] `openspec/changes/0002-route-duplicate/tasks.md` 回填完成状态与验证记录

## 7. 发布门禁

- [x] `cd core/admin/web && pnpm test`（含新增单测，全绿）
- [x] `cd core/admin/web && pnpm build`（`tsc -b` + vite，通过）
- [x] `cd core/admin/web && pnpm lint`（新增代码无新增 lint 错误，能过则过）
- [x] `go test ./core/...` 回归（无 Go 变更；环境限制用例参照 0001 记录）
- [x] 手工冒烟：两条入口 + 冲突拦截 + 插入位置 + 撤销 + 保存发布 diff（必要时本地起 Admin）
- [x] `version.go` → `1.23.1`，提交 `chore(release): bumped version to v1.23.1`，打 tag `v1.23.1`
- [x] 推送 `master` 与 tag 到 `origin`（https://github.com/go-zoox/ingress）

## Verification

（实现完成后回填：命令、结果、日期、版本）

## Verification（2026-11-04，v1.23.1）

- `cd core/admin/web && pnpm test`：18 个测试文件 / 141 用例通过（新增 `configEntities.copy.test.ts` 18 例）。
- `cd core/admin/web && pnpm exec tsc -b`：通过；`pnpm build`（`tsc -b` + vite）通过。
- `pnpm exec eslint`（本次改动文件）：无新增错误；仓库基线仍有 83 个历史 lint error（`react-hooks/set-state-in-effect` 等），`pnpm lint` 整体仍非绿色，属既有问题。
- 真实 Admin UI 冒烟：本机 `go run -tags adminui` + 临时配置（18080/19080，SQLite 落在 /tmp），headless chromium + CDP 驱动，**32 项断言全过**：
  - 行内「高级」菜单第一项「复制路由」→ 弹窗预填 `api-copy.example.com`，横幅展示源规则 #1 与复制说明；
  - Host 改为 `admin.example.com` → 字段级错误含「规则 #2」，创建按钮禁用；
  - 创建 → 副本紧随源规则（第 2 行）、行高亮、撤销条、未保存标记；撤销后恢复 5 条；
  - 再次创建 → 「保存与发布」diff 含新增 host（点击取消，未写盘）；
  - 详情页「复制路由」→ `/routes?copy=0` 打开同一弹窗且 `copy` 参数已清理；
  - 正则规则复制 → Host 不改名 + 冲突提示 + 创建禁用；
  - 全程无未捕获 JS 异常。
- `go test ./core/...`：本 change 无 Go 代码改动；环境相关历史失败（root 下的 geoip「不可读文件」用例、`access_log_empty` 15m 用例）与本次无关，其余包通过。
- 原型回归：`prototypes/route-copy/` → `node scripts/smoke.js` 通过；`node scripts/ui-smoke.mjs` 33 项通过（headless chromium）。
- 发布：`version.go` → `1.23.1`，`chore(release): bumped version to v1.23.1`，tag `v1.23.1`，推送 `origin master` 与 tag。
