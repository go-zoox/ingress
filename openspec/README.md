# openspec —— 轻量规格（无 CLI）

借鉴 OpenSpec 格式，管住「先对齐、再写码」。本仓库只保留最小结构，不依赖 CLI。

- `specs/<capability>/spec.md`：**当前能力的事实描述**（系统现在是什么样）。
- `changes/<id>-<slug>/`：一次需求的四件套：
  - `proposal.md`：为什么做、做什么、不做什么、影响面、验收；
  - `design.md`：怎么做（关键算法、配置映射、边界与风险）；
  - `tasks.md`：实现步骤与验收清单；
  - `specs/<capability>/spec.md`：本次新增 / 修改的 Requirement 增量（`ADDED` / `MODIFIED`）。
- 完成后：增量合入 `specs/`，change 目录移入 `changes/archive/`。

命名：`<4 位序号>-<动词>-<范围>`，例如 `0001-route-ip-allowlist`。
