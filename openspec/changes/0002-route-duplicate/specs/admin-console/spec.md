# Admin Console — spec delta（0002-route-duplicate）

> 本文件是本次变更的 **Requirement 增量**。实现完成、验收通过后，把以下 Requirement 合入
> `openspec/specs/admin-console/spec.md`，并归档 change 目录。

## ADDED Requirements

### Duplicate route action

管理后台 SHALL 提供「复制路由」操作，允许用户基于一条已有路由创建副本：

- 入口 SHALL 至少包含两处：路由规则列表行内「高级」菜单的第一项，以及路由详情页页头按钮；
- 复制 SHALL 深拷贝源规则的全部配置（`backend`、`paths`、`connect_proxy`、`rate_limit`、`security`、`waf` 等），副本与源规则 SHALL 互相独立；
- 复制弹窗 SHALL 复用规则编辑表单，标题为「复制路由」，并展示源规则编号与 Host；
- 副本 SHALL 作为普通 `rules[]` 行参与保存、diff 与发布，SHALL NOT 引入新的配置语义或后端接口；
- 副本的启用状态 SHALL 默认跟随源规则，并允许用户在弹窗内修改。

### Copy host suggestion and duplicate guard

复制弹窗 SHALL 生成并校验副本 Host：

- 默认 Host SHALL 为源 Host 加 `-copy`（第一个 label，通配符保留 `*.` 前缀、只改 `*` 后的第一个 label）；
- 目标 Host 已存在时 SHALL 顺延为 `-copy-2`、`-copy-3`…（大小写不敏感）；已有 `-copy-N` 后缀时 SHALL 继续递增；
- 正则 Host SHALL NOT 自动改名，SHALL 展示人工确认提示；
- Host 为空、包含空白字符、或与任意既有规则 Host 重复时 SHALL 在弹窗内报错并禁用创建；错误 SHALL 指明冲突规则编号与 Host；
- 复制模式下的重复 Host 拦截 SHALL NOT 改变编辑模式既有行为。

### Copy placement and post-create feedback

- 副本插入位置 SHALL 默认为源规则之后，并 SHALL 允许用户选择列表末尾；
- 创建后列表 SHALL 高亮新行，SHALL 出现可撤销提示（恢复复制前的规则草稿），并沿用现有「未保存」状态；
- 撤销 SHALL 只影响未保存草稿，SHALL NOT 影响已保存或已发布配置。

### Deep link for copy from route detail

- 路由详情页的复制入口 SHALL 通过 `/routes?copy=<ruleIndex>` 打开列表页的复制弹窗；
- 弹窗打开后 SHALL 清理 URL 中的 `copy` 参数，避免刷新或后退重复打开；
- doc 未加载完成或下标越界时 SHALL 安全降级（不弹空弹窗、不崩溃）。
