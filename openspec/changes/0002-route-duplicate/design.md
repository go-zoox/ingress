# design: 0002-route-duplicate

> 结论先行：**不新增接口与配置语义，复制是 `rules` 模块文档上的「深拷贝 + 改 Host + 选择插入位置」**。弹窗复用现有规则编辑分区表单，把「要不要改 Host」变成唯一强校验项，并用插入位置保证列表顺序（匹配优先级）可控。

## 1. 数据流与状态机

```text
RouteRulesManageTab / ConfigModuleForm（rules 模块 doc）
        │  RulesEditor（doc + onChange）
        │
        ├─ openCopy(i)
        │     源规则 = rulesFromDoc(doc)[i]
        │     form   = ruleToForm(源规则)          # 还原成可编辑表单
        │     host   = suggestCopyHost(源规则.host, 现有 hosts, 源规则.host_type)
        │     mode   = 'copy' + sourceIndex + position
        │
        ├─ 用户改 Host / 插入位置 / 其余字段
        │     Host 实时校验：空 / 空白 / 与既有规则重复 → 内联错误 + 禁用「创建副本」
        │
        └─ save()
              origin = deepClone(源规则)
              row    = formToRule(form, origin)   # 保留表单未覆盖的键（其他 waf / backend 键）
              rules  = insertCopiedRule(rules, sourceIndex, position, row)
              onChange({ rules })
              → dirty（现有 doc diff）+ 行高亮 + 可撤销提示
              → 现有「保存与发布」链路落盘
```

关键点：

- **不引入新状态字段**：副本就是普通 `rules[]` 行，YAML 结构与手工新增完全一致；dirty / diff / 发布全部复用现有能力。
- **复制模式不写入源规则**：只读 `rules[sourceIndex]`，新数组替换，源对象不动。
- **撤销**：保留复制前的 `rules` 数组快照；提示条点击「撤销」时 `onChange({ rules: snapshot })`。只作用于未保存草稿。

## 2. 副本 Host 生成算法

```text
effectiveType =
  显式 host_type != auto ？host_type
  : Host 含正则元字符 ( ) [ ] { } ^ $ | + ? \ ？regex
  : Host 含 * ？wildcard
  : exact

if effectiveType == regex:
    host = 原 Host（不自动改名；弹窗提示人工确认）
else:
    prefix = (wildcard 且 Host 以 "*." 开头) ? "*." : ""
    body   = 去掉 prefix 的 Host
    base   = patchFirstLabel(body, 0)
    candidate = prefix + base
    while candidate ∈ 现有 Host（大小写不敏感）且 尝试次数 < 20:
        candidate = prefix + patchFirstLabel(body, attempt++)

patchFirstLabel(body, attempt):
    labels = body.split(".")
    取第一个非空 label
    解析已有 -copy / -copy-N 后缀 → existingLevel
    level = existingLevel + attempt
    label = 去掉后缀的 basename
    label = basename + (level == 0 ? "-copy" : "-copy-" + (level + 1))
```

示例：

| 源 Host | 结果 |
|---|---|
| `api.example.com` | `api-copy.example.com` |
| `admin.example.com`（`admin-copy.example.com` 已存在） | `admin-copy-2.example.com` |
| `*.shop.example.com` | `*.shop-copy.example.com` |
| `api-copy.example.com` | `api-copy-2.example.com` |
| `^legacy\.example\.com$` | 原样，`auto: false`，提示人工改 |

> 与 ingress 编译期 Host 类型推断保持一致（`core/compile.go`：正则元字符优先于 `*`，否则 exact）。显式 `host_type` 优先，避免把 `host_type: exact` 的正则样字符串误判为 regex。

## 3. Host 冲突校验

```text
normalize(host) = trim + toLowerCase
conflict = 现有 rules[] 中第一条 normalize(host) 相同的规则（含复制源自身）
error:
  - Host 为空          → "Host 不能为空"
  - 含空白字符          → "Host 不能包含空白字符"
  - 命中冲突             → "Host 与规则 #N（host）重复；列表顺序在前的先匹配，副本通常不会命中"
```

- 只拦截**复制模式**；编辑模式保持现状（避免把「同 Host 多规则 + path 区分」的既有用法一刀切禁掉）。
- 正则 Host 不改名时天然会与源规则重复 → 被这条校验拦住，正好达到「必须人工确认」的目的。
- 校验是客户端产品约束；后端 `ingress validate` 语义不变（重复 Host 合法但会被前序规则遮蔽）。

## 4. 插入位置与启用状态

- `position: 'after' | 'end'`，默认 `after`：
  - `after` → `sourceIndex + 1`；
  - `end` → `rules.length`；
  - 插入后新行下标 = 插入点（用于高亮与提示文案）。
- 副本 `enabled` 跟随源规则（表单里的启用开关预填源值，用户仍可改）。
- 列表顺序 = 匹配优先级；`after` 让副本与源规则相邻，便于对照与调整优先关系。YAML 数组顺序即列表顺序，保存链路无需特殊处理。

## 5. 深拷贝与字段保真

```text
origin = JSON.parse(JSON.stringify(源规则))     # 断开引用，保证副本独立
row    = formToRule(form, origin)                # form 覆盖受管字段，origin 保留未受管键
```

`formToRule(form, original)` 的既有语义是「保留表单未覆盖的键」：

- `next = { ...original }`，然后写入受管字段；
- `backend = formToBackend(form, original.backend)` 保留后端中表单未覆盖的键；
- `waf` 用 `buildIPAllowlistWaf(form, original.waf)` 合并，保留自定义 `waf.rules` 等；
- `paths` 用 `formToPath(p, original.paths[i])` 保留 Path 级未覆盖键。

因此复制必须传入源规则（深拷贝后）作为 `original`，而不是 `formToRule(form)`；否则自定义 WAF / 未识别字段会丢。单测覆盖：复制后 `waf.rules`、`backend` 未知键、Path 级配置仍在，且修改副本不影响源对象。

## 6. UI 组成

### 6.1 弹窗

```text
ConfigEntityModal（wide, title = 复制路由）
├─ 复制横幅（仅复制模式）
│  ├─ 源规则 #N · host / 副本 badge
│  ├─ 说明：除 Host 外全部配置已复制；未保存未发布不影响线上
│  ├─ 插入位置 select（紧随 #N 之后 / 列表末尾）
│  └─ Host 提示：自动生成说明 / 冲突顺延说明 / 正则不改名警告
└─ RuleEntityFormSections（复用全部分区；Host 字段支持 error 展示）
```

- 基础信息分区里 Host 字段旁展示字段级错误（新增 `FormField error?: string`，`aria-invalid` + `.form-error-text`）；
- 禁用保存条件：`ruleSaveDisabled(draft) || copyHostError`；
- 标题、横幅文案沿用原型 `prototypes/route-copy/`。

### 6.2 创建后的反馈

- 新行加 `row-flash`（约 3.2s，`highlightIndex` 状态）；
- 固定底部提示条：`已创建副本 <host>（第 N 条） [撤销] [×]`，8s 自动消失；
- 撤销 = `onChange({ rules: 快照 })`，同时清除高亮；
- 不新增全局 Toast API 改造（Toast 目前只支持 message/type，避免影响其它页面）。

### 6.3 入口

| 入口 | 实现 |
|---|---|
| 行内「高级 ▾」 | `EntityRowActions` 的 `menuItems` 第一项「复制路由」→ `openCopy(i)` |
| 路由详情页 | `PageHeader.actions` 新增按钮 → `navigate('/routes?copy=<ruleIndex>')` |
| 深链 | `RoutesPage` 读取 `copy` 查询参数 → `RouteRulesManageTab` 在 doc 加载完成后调用 `rulesEditorRef.current.openCopy(ri)`，随后用 `replace` 清掉参数（避免刷新/后退重复弹窗） |

`RulesEditorHandle` 增加 `openCopy(index: number): void`；Config 页的 rules 编辑模块无需额外改动即可用同一个菜单入口。

## 7. 边界与风险

| 场景 | 行为 |
|---|---|
| 源规则禁用 | 副本禁用；仍可创建 |
| 源规则为 regex Host | Host 原样带入 + 警告；不改名必与源冲突 → 创建按钮禁用，提示手改 |
| 源规则为 wildcard | 保留 `*.`，改后段第一个 label |
| 源规则 host_type 显式 exact | 按 exact 处理（即使字符串含正则符号） |
| 列表为空 | 无行可复制，入口自然不存在 |
| 连续复制同一行 | 每次都基于当前最新列表生成区分名（-copy-2/-copy-3…） |
| 复制后用户再次拖拽排序 | 现有拖拽逻辑按数组重排，副本作为普通行参与 |
| 复制后未保存就刷新 | 草稿丢失（与现有编辑一致） |
| 撤销后再复制 | 提示条被新提示替换；快照基于最新列表 |

风险与缓解：

- **formToRule 合并行为变化**：若未来调整受管键集合，复制路径需同步单测；本 change 只依赖其「保留未覆盖键」的既有契约；
- **Host 冲突误伤**：只在复制模式强制，编辑模式不提；用户确有同 Host 多规则需求时仍可走编辑/YAML；
- **深拷贝开销**：规则对象很小，JSON clone 可忽略；
- **撤销语义**：只回滚草稿，不影响已保存/已发布配置，文案需明确「未保存」。

## 8. 验证策略

- 纯函数单测（vitest）：`configEntities.copy.test.ts`——Host 生成（exact/wildcard/regex/显式类型/占用顺延/副本的副本）、冲突检测（大小写、排除自身）、插入位置、深拷贝保真；
- 构建门禁：`pnpm test`、`pnpm build`（tsc -b + vite）；
- 手工冒烟：原型对照走查 6 条验收路径；必要时本地起 Admin（`go run -tags adminui ./cmd/ingress run -c examples/admin-console/ingress.yaml`）验证保存/发布 diff；
- 无 Go 变更，`go test` 仅作为回归（已有环境限制见 0001 的 Verification）。
