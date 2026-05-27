# Web 应用防火墙（WAF）

Ingress 在 **路由匹配完成之后**、**重定向 / 静态处理 / 回源代理之前**，对请求执行第一代 WAF：

- **IP**：`deny` 先于 `allow`；若配置了非空 `allow`，则仅列表内网段可通过 IP 阶段。
- **特征**：针对 `path`、`query`、`uri`（path + raw query）、`headers`（所有头拼接）或 `header:Name` 做单次匹配；支持 `regex` / `contains`。
- **可选内置规则**：常见 SQL 注入试探、路径穿越、脚本相关试探；用 **`disable_builtin: true`** 关闭。

默认 **不扫请求体**。上线前先用全局或单条的 **`log_only`** 审计。

顶层 **`waf:`** 为类型化基线；**`rules[].waf`** 为 **局部 YAML map**，在 `config.Load` 之后由 **`waf.ApplyRulePatchesFromYAML`**（`core/waf/yaml.go`）按字段合并（解码器无法用同一流程表达“只覆盖部分字段”）。

评估顺序：**合并策略 → enabled → 客户端 IP → deny/allow → 特征**。`trust_proxy` 为真时用 **`X-Forwarded-For`**，**`xff_index`** 选段（0 为最左一段，负数从末尾倒数）。

## 内置 starter 规则

未设置 **`disable_builtin: true`** 时，会先加载下表中的规则，再追加你在 `waf.rules` 里自定义的规则。内置 **`id`** 固定，便于在日里识别，也可用 **同名 `id`** 在配置里**覆盖**内置定义（示例：`examples/waf/rule-merge-by-id.yaml`）。正则语义为 [Go `regexp`](https://pkg.go.dev/regexp/syntax)。

| ID | 检测目标（`targets`） | 说明 |
|----|----------------------|------|
| `builtin:sqli-common` | `uri`（path + query） | 常见 SQL 注入试探：`union select`、`sleep(`、`benchmark(`、`; drop/truncate/alter table` 等 |
| `builtin:path-traversal` | `path` | 路径穿越：`../`、`..\`、编码形式的 `..`、`etc/passwd` 等 |
| `builtin:xss-lite` | `uri` | 轻度 XSS 试探：`<script`、`javascript:`、`on…=` 类事件属性 |

与代码一致的正则（源码：`core/waf/builtin.go`）：

```
builtin:sqli-common
(?is)(union\s+select\b|sleep\s*\(|benchmark\s*\(|;\s*(drop|truncate|alter)\s+table\b)

builtin:path-traversal
(?:\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\\\|etc/passwd\b)

builtin:xss-lite
(?is)(<\s*script\b|javascript:\s*|\bon[a-z]+\s*=)
```

内置规则可能对正常流量误报，可先用 **`log_only`** 观察，或 **`disable_builtin: true`** 后仅靠自定义规则。

可运行样例：`examples/waf/`。字段表格见 **[配置参考](./configuration.md)**；英文版说明：**[English WAF](../../guide/waf.md)**。
