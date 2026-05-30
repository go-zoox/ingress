# Scenarios 配置方案

**已选：方案 C** — `scenarios.active` + `scenarios.items[]` 显式列表。

示例：[`examples/scenarios/design-option-c-list.yaml`](../../examples/scenarios/design-option-c-list.yaml)

Admin API：
- `GET /api/v1/scenarios` — 当前场景与列表
- `PUT /api/v1/scenarios/active` — body `{ "id": "live" }`，写回 YAML 并重载

---

## 历史方案对比（选型记录）

目标：在**一份** `ingress.yaml` 里定义多个运行场景（日常 / 直播等），通过 `scenario` 选择当前生效场景；Admin Console 读取场景列表与当前值，支持切换并重载。

---

## Admin Console 预期 API（实现阶段）

| 接口 | 用途 |
|------|------|
| `GET /api/v1/scenarios` | 返回 `{ active, scenarios: [{ id, label, description?, ... }] }` |
| `PUT /api/v1/scenarios/active` |  body `{ id: "live" }` → 写回 YAML 的 `scenario` 并 reload |

配置层需稳定提供：**当前场景 id** + **可枚举的场景列表** + **每个场景的展示名**。

---

## 方案 A（推荐）：`scenario` + `scenarios` 映射 + `overlay`

根配置保留**共享基线**（`port`、`https`、`admin`、`rules` 完整路由等）。每个场景只写**差异**，放在 `overlay` 里；运行时按 `scenario` 合并后再 `validate` / `prepare`。

```yaml
scenario: daily   # 当前场景；可被环境变量 INGRESS_SCENARIO 覆盖（实现时可选）

scenarios:
  daily:
    label: 日常
    description: 直连原站，商品读接口不缓存
    overlay:
      cache:
        ttl: 60
      rules:
        - host: shop.example.com
          backend:
            cache:
              enabled: false

  live:
    label: 直播
    description: 高并发，商品列表/详情缓存，写路径 bypass
    overlay:
      cache:
        host: redis.internal
        port: 6379
        prefix: "ingress:live:"
      rules:
        - host: shop.example.com
          backend:
            cache:
              enabled: true
              default: bypass
              paths: [...]
```

**合并规则（建议）**

| 顶层键 | 行为 |
|--------|------|
| `cache`、`rate_limit`、`waf`、`maintenance`、`security` | overlay 字段**浅覆盖**基线同名字段 |
| `rules` | 按 **`host` 精确匹配**基线规则，对匹配项 **deep-merge** `backend` / `paths` / `waf` 等；overlay 中**新 host** 追加到 rules 末尾 |
| 未出现在 overlay 的键 | 保持基线不变 |

**优点**：基线一份完整路由；场景只维护差异；适合 Admin「切换场景 = 改 `scenario` + reload」。  
**缺点**：需实现 host 级 deep-merge（与 WAF patch 类似，范围更大）。

示例文件：[`examples/scenarios/design-option-a-overlay.yaml`](../../examples/scenarios/design-option-a-overlay.yaml)

---

## 方案 B：场景内直接写模块（无 `overlay` 包裹）

与 A 相同语义，但去掉 `overlay` 一层，场景块顶格写可覆盖的顶层键：

```yaml
scenario: daily

scenarios:
  daily:
    label: 日常
    description: ...
    cache:
      ttl: 60
    rules:
      - host: shop.example.com
        backend:
          cache:
            enabled: false

  live:
    label: 直播
    description: ...
    cache:
      host: redis.internal
    rules:
      - host: shop.example.com
        backend:
          cache: { enabled: true, ... }
```

**元数据键**（不参与 merge，仅 Admin 展示）：`label`、`description`（可选 `icon`、`order`）。

**优点**：YAML 更短。  
**缺点**：场景块里「元数据」与「可覆盖配置」混在一起，扩展 `label` 等同名冲突风险（需保留字列表）。

示例文件：[`examples/scenarios/design-option-b-flat.yaml`](../../examples/scenarios/design-option-b-flat.yaml)

---

## 方案 C：显式列表 + `active`（偏 API 友好）

```yaml
scenarios:
  active: daily
  items:
    - id: daily
      label: 日常
      description: 直连原站
      overlay:
        rules: [...]
    - id: live
      label: 直播
      description: 高并发缓存
      overlay:
        cache: {...}
        rules: [...]
```

**优点**：`items[]` 顺序即 Admin 下拉顺序；`id` 显式，不依赖 map 键。  
**缺点**：与 ingress 其它块（`cache:`、`rules:`）风格不一致；`scenarios.active` 与顶层 `scenario` 二选一易混淆。

示例文件：[`examples/scenarios/design-option-c-list.yaml`](../../examples/scenarios/design-option-c-list.yaml)

---

## 方案 D：仅场景目录（不在 YAML 内嵌 overlay）

```yaml
scenario: live
scenarios:
  daily: { label: 日常, file: scenarios/daily.yaml }
  live:   { label: 直播, file: scenarios/live.yaml }
```

**优点**：大场景 diff 独立文件，Git diff 清晰。  
**缺点**：Admin 编辑/切换需读写多文件；`validate` 路径复杂；与「单文件 ingress.yaml」Admin 模型冲突。

---

## 对比摘要

| | A overlay | B flat | C list | D 外置文件 |
|--|-----------|--------|--------|------------|
| Admin 列表/当前 | ✅ | ✅ | ✅ 顺序最好 | ✅ |
| 单文件编辑 | ✅ | ✅ | ✅ | ❌ |
| 与现有 YAML 风格 | ✅ | ✅ | △ | △ |
| 实现复杂度 | 中 | 中 | 中 | 高 |
| 场景 diff 可读性 | ✅ | ✅ | ✅ | ✅✅ |

**推荐：方案 A**（`overlay` 区分元数据与配置）；若团队强烈偏好少嵌套，选 **方案 B**。

---

## 共用约定（无论选哪种）

1. **`scenario` 省略**：不应用任何场景 overlay，与 today 行为一致（向后兼容）。
2. **`scenario` 无效 id**：`ingress validate` / 启动失败，报错列出合法 id。
3. **环境变量**（可选）：`INGRESS_SCENARIO=live` 覆盖 YAML 中的 `scenario`（便于 K8s 临时切场景，Admin 仍以文件为准或显示 override）。
4. **Admin 模块**：`scenarios` 单独成 Admin 配置模块「场景」；`scenario`（当前）可在该模块或基础模块编辑。
5. **rules 合并**：仅 patch **已存在于基线** 的 host，避免场景 accidentally 删掉其它路由（除非显式允许 `rules_replace: true`，默认关闭）。

---

## 请选择

回复选项即可，例如：**「A + 环境变量覆盖」** 或 **「B，不要 overlay」**。

选定后下一步：Go 类型定义、`ApplyScenario` 合并、`validate` 测试、Admin `GET/PUT scenarios` API。
