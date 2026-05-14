# Web Application Firewall (WAF)

Ingress can run a **first-version WAF** on matched routes **after routing** and **before** redirects, handlers, or upstream proxies. It covers:

- **IP lists** (`deny` then optional `allow` gate).
- **Signature checks** against `path`, `query`, assembled `uri`, all header lines (`headers`), or one header (`header:User-Agent`).
- An optional **embedded starter ruleset** (SQLi/path traversal/reflected-scripting probes). Turn off with **`disable_builtin: true`** if it is too noisy.

There is **no request-body scanning**. Use **`log_only`** (global or per rule) to audit before blocking.

## Enabling WAF

- Top-level **`waf:`** is the typed global baseline.
- **`rules[].waf`** is a partial YAML map merged onto that baseline for the matched host rule.

::: warning Config loader caveat
The generic config decoder cannot partially merge map keys into structs, so **`rules[].waf`** is applied in **`waf.ApplyRulePatchesFromYAML`** (`core/waf/yaml.go`) after `config.Load`. Embedded configs built in Go must set **`rules[i].WAFPatch`** manually when you need route overlays without a file.
:::

## Evaluation order

1. Effective policy = merge(global `waf`, `rules[i].WAFPatch`).
2. Exit if `enabled` is false.
3. Resolve client IP ( **`trust_proxy`** + **`xff_index`** ; default index **0** = leftmost segment in `X-Forwarded-For` ).
4. **Deny** list, then **Allow** when non-empty (only listed nets may pass the IP phase).
5. Signatures: starters unless `disable_builtin`, then custom rules in merged order.

Blocked clients receive **`block_status_code`** (default **403**), **`block_content_type`**, and **`block_body`**.

## Built-in starter rules

When **`disable_builtin`** is not `true`, the following rules are appended **before** custom entries in `waf.rules`. IDs are stable so you can **replace** a builtin by adding your own rule with the **same `id`** (see [`examples/waf/rule-merge-by-id.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/waf/rule-merge-by-id.yaml)). Patterns use [Go `regexp`](https://pkg.go.dev/regexp/syntax) syntax.

| ID | Targets | Description |
|----|---------|-------------|
| `builtin:sqli-common` | `uri` | Common SQL-injection style probes in path + raw query (`union select`, `sleep(`, `benchmark(`, `; drop/truncate/alter table`, …). |
| `builtin:path-traversal` | `path` | Path traversal probes (`../`, `..\`, encoded `..`, `etc/passwd`). |
| `builtin:xss-lite` | `uri` | Light reflected-scripting probes (`<script`, `javascript:`, `on*=`-style event handlers). |

Exact patterns (from [`core/waf/builtin.go`](https://github.com/go-zoox/ingress/blob/master/core/waf/builtin.go); change there if you fork):

```
builtin:sqli-common
(?is)(union\s+select\b|sleep\s*\(|benchmark\s*\(|;\s*(drop|truncate|alter)\s+table\b)

builtin:path-traversal
(?:\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\\\|etc/passwd\b)

builtin:xss-lite
(?is)(<\s*script\b|javascript:\s*|on\w+\s*=)
```

Starters can false-positive on unusual but legitimate traffic — use **`log_only`** or **`disable_builtin: true`** and replace with stricter custom rules as needed.

## Custom rules (`waf.rules`)

| Field | Notes |
|-------|-------|
| `id` | Required. Route-level `waf.rules` with the same `id` **replace** the global rule of that id. |
| `type` | `regex` (default) or `contains` |
| `pattern` | Compiled at startup for `regex`. |
| `targets` | One or more of `path`, `query`, `uri`, `headers`, `header:…` |

Runnable files: [`examples/waf/`](https://github.com/go-zoox/ingress/tree/master/examples/waf).

## See also

- [Configuration reference](./configuration.md)
- [WAF examples](../examples/waf.md)
