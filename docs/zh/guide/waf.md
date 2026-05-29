# Web 应用防火墙（WAF）

Ingress 在 **路由匹配完成之后**、**重定向 / 静态处理 / 回源代理之前**，对请求执行第一代 WAF：

- **IP**：`deny` 先于 `allow`；若配置了非空 `allow`，则仅列表内网段可通过 IP 阶段。
- **特征**：针对 `path`、`query`、`uri`（path + raw query）、`headers`（所有头拼接）或 `header:Name` 做单次匹配；支持 `regex` / `contains`。
- **可选内置规则**：常见 SQL 注入试探、路径穿越、脚本相关试探；用 **`disable_builtin: true`** 关闭。

默认 **不扫请求体**。每条签名规则（含内置）可设 **`action`**：`block`（拦截，默认）、`audit`（只记录）、`pass`（命中后放行并跳过后续签名）。仍可用全局或单条 **`log_only: true`**（等价于未写 `action` 时的 `audit`）。

顶层 **`waf:`** 为类型化基线；**`rules[].waf`** 为 **局部 YAML map**，在 `config.Load` 之后由 **`waf.ApplyRulePatchesFromYAML`**（`core/waf/yaml.go`）按字段合并（解码器无法用同一流程表达“只覆盖部分字段”）。

评估顺序：**合并策略 → enabled → 客户端 IP → deny/allow → 特征**。`trust_proxy` 为真时用 **`X-Forwarded-For`**，**`xff_index`** 选段（0 为最左一段，负数从末尾倒数）。

## 内置 starter 规则

未设置 **`disable_builtin: true`** 时，会先加载下表中的规则，再追加你在 `waf.rules` 里自定义的规则。内置 **`id`** 固定，便于在日里识别，也可用 **同名 `id`** 在配置里**覆盖**内置定义（示例：`examples/waf/rule-merge-by-id.yaml`）。正则语义为 [Go `regexp`](https://pkg.go.dev/regexp/syntax)。

| ID | 检测目标（`targets`） | 说明 |
|----|----------------------|------|
| `builtin:sqli-common` | `uri`（path + query） | 常见 SQL 注入试探：`union select`、`sleep(`、`benchmark(`、`; drop/truncate/alter table` 等 |
| `builtin:path-traversal` | `path` | 路径穿越：`../`、`..\`、编码形式的 `..`、`etc/passwd` 等 |
| `builtin:xss-lite` | `uri` | 轻度 XSS 试探：`<script`、`javascript:`、`on…=` 类事件属性 |
| `builtin:rce-probes` | `uri` | 命令注入/RCE 探针 |
| `builtin:jndi-lookup` | `uri`、`headers` | JNDI `${…jndi:` 注入 |
| `builtin:sensitive-files` | `path` | 敏感路径（`.env`、`.git/`、管理后台等） |
| `builtin:ssrf-probes` | `uri` | 云元数据 / `file://` / `gopher://` SSRF 探针（不含 localhost，避免 OAuth 误报） |
| `builtin:scanner-ua` | `header:User-Agent` | 常见扫描器 UA |
| `builtin:crlf-injection` | `uri`、`headers` | CRLF / 响应拆分 |
| `builtin:php-ssti` | `uri` | PHP eval、`php://` 等 |

单条启停：

- **`disable_builtin: false`**（默认）：内置规则默认全开，可在 **`waf.builtin_rules`** 里对某条设为 `false`。
- **`disable_builtin: true`**：内置规则默认全关，可在 **`waf.builtin_rules`** 里单独设为 `true`。
- 自定义 **`waf.rules[]`** 支持 **`enabled: false`**（省略时默认启用）。

示例：

```yaml
waf:
  enabled: true
  builtin_rules:
    builtin:scanner-ua: false
  rules:
    - id: my-rule
      enabled: false
      type: contains
      pattern: /internal
      targets: [path]
```

与代码一致的正则（源码：`core/waf/builtin.go`）：

```
builtin:sqli-common
(?is)(union\s+select\b|sleep\s*\(|benchmark\s*\(|;\s*(drop|truncate|alter)\s+table\b)

builtin:path-traversal
(?:\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\\\|etc/passwd\b)

builtin:xss-lite
(?is)(<\s*script\b|javascript:\s*[a-z]|\bon(?:click|load|error|focus|blur|change|submit|mouse\w*|key\w*|touch\w*|pointer\w*|scroll|dblclick|drag\w*|drop|input|reset|select|wheel|copy|cut|paste|abort|contextmenu|message|unload|beforeunload)\s*=)
```

内置规则可能对正常流量误报，可先用 **`log_only`** 观察，或 **`disable_builtin: true`** 后仅靠自定义规则。常见正常流量回归见 **`core/waf/builtin_false_positive_test.go`**。

可运行样例：`examples/waf/`。字段表格见 **[配置参考](./configuration.md)**；英文版说明：**[English WAF](../../guide/waf.md)**。
