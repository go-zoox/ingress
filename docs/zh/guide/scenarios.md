# 运行场景

Ingress 可在**一份** `ingress.yaml` 中定义多个**命名 overlay 场景**。根配置为**基线**（日常流量）；切换 `scenarios.active` 即可合并 overlay，用于直播、演练等模式，无需维护多份完整配置文件。

## 配置结构（方案 C）

```yaml
scenarios:
  active: default   # 或 live、drill 等
  items:
    - id: live
      label: 直播
      description: 高并发下缓存商品读接口
      overlay:
        cache:
          host: redis.internal
          prefix: "ingress:live:"
        rules:
          - host: shop.example.com
            backend:
              cache:
                enabled: true
                default: bypass
                paths:
                  - match: /api/v1/products
                    match_type: prefix
                    action: cache
                    ttl: 60
```

| 字段 | 说明 |
|------|------|
| `scenarios.active` | 当前场景 id。**`default`**（或留空）= 根配置，**不应用 overlay**。 |
| `scenarios.items[]` | 仅 overlay 场景（`id`、`label`、`description`、`overlay`）。 |
| `items[].overlay` | 在 prepare/reload 时合并到基线上的差异块。 |

### 保留的 `default` 场景

- **`scenarios.active: default`** 表示直接使用根 `ingress.yaml`，不合并 overlay。
- **不要**在 `items[]` 中创建 `id: default` 的条目（系统保留）。
- Admin 控制台列表首项固定为虚拟 **默认** 场景。

### Overlay 合并规则

支持的 overlay 键：`cache`、`rate_limit`、`waf`、`maintenance`、`security`、`rules`。

- 顶层键对基线同名字段 **deep-merge**。
- `rules[]` 按 **`host`** 匹配并合并到已有路由（未知 host 校验失败）。

### 运行时覆盖

环境变量 **`INGRESS_SCENARIO=<id>`** 可覆盖 YAML 中的 `scenarios.active`（容器部署常用）。

```bash
INGRESS_SCENARIO=live ingress run -c ingress.yaml
```

## 校验与切换

```bash
ingress validate -c examples/scenarios/ingress.yaml
ingress run -c examples/scenarios/ingress.yaml
```

切换并 reload：

- 修改 `scenarios.active` 后 `SIGHUP` / Admin **发布**
- Admin API：**`PUT /api/v1/scenarios/active`**，body `{ "id": "live" }`

## Admin 控制台

**`admin.enabled: true`** 时，打开 **维护 → 场景管理**：

- overlay 场景增删改查、选择 **当前场景**、**保存与发布**
- **切换生效** 写回 `scenarios.active` 并热加载

**配置 → 场景** 模块编辑同一份 YAML。

## 示例

可运行样例：[`examples/scenarios/`](https://github.com/go-zoox/ingress/tree/master/examples/scenarios) — 见 [场景示例](../examples/scenarios.md)。

方案说明：[`docs/plans/2026-05-30-scenarios-config-design.md`](https://github.com/go-zoox/ingress/blob/master/docs/plans/2026-05-30-scenarios-config-design.md)。
