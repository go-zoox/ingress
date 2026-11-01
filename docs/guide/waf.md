# Web Application Firewall (WAF)

Ingress can run a **first-version WAF** on matched routes **after routing** and **before** redirects, handlers, or upstream proxies. It covers:

- **IP lists** (`deny` then optional `allow` gate).
- **Route-level IP allowlist** (`rules[].waf.allow` per host; see [Route-level IP allowlist](#route-level-ip-allowlist-per-host)).
- **Host allowlist** (`allow_hosts`): listed hosts skip all WAF phases (IP lists and signatures).
- **Per-rule host allowlist** (`rules[].allow_hosts`): listed hosts skip **that rule only**; other rules still run.
- **Signature checks** against `path`, `query`, assembled `uri`, all header lines (`headers`), or one header (`header:User-Agent`).
- An optional **embedded starter ruleset** (SQLi/path traversal/reflected-scripting probes). Turn off with **`disable_builtin: true`** if it is too noisy.

There is **no request-body scanning**. Each signature rule (custom or built-in) can **`block`** (default), **`audit`** (log only), or **`pass`** (treat match as allowed and skip remaining signatures). Global or per-rule **`log_only: true`** is still supported and equals **`action: audit`** when `action` is omitted.

## Enabling WAF

- Top-level **`waf:`** is the typed global baseline.
- **`rules[].waf`** is a partial YAML map merged onto that baseline for the matched host rule.

::: warning Config loader caveat
The generic config decoder cannot partially merge map keys into structs, so **`rules[].waf`** is applied in **`waf.ApplyRulePatchesFromYAML`** (`core/waf/yaml.go`) after `config.Load`. Embedded configs built in Go must set **`rules[i].WAFPatch`** manually when you need route overlays without a file.
:::

## Evaluation order

1. Effective policy = merge(global `waf`, `rules[i].WAFPatch`).
2. Exit if `enabled` is false.
3. **`allow_hosts`**: skip all WAF when request Host matches (exact, `*` wildcard, or auto-detected Go regex — same inference as ingress host routing).
4. Resolve the client IP. Precedence: **`trusted_proxies`** → **`trust_proxy` + `xff_index`** → TCP peer. With **`trusted_proxies`**, `X-Forwarded-For` is trusted only when the TCP peer is inside those networks, and the client is the **rightmost XFF hop that is not itself a trusted proxy**. The WAF always starts from the real TCP peer, never from a `RemoteAddr` that forwarding headers may have rewritten.
5. **Deny** list, then **Allow** when non-empty (only listed nets may pass the IP phase).
6. Signatures: starters unless `disable_builtin`, then custom rules in merged order. Before each signature, **`rules[].allow_hosts`** on that rule skips it when Host matches (same pattern syntax as global `allow_hosts`).

Blocked clients receive **`block_status_code`** (default **403**), **`block_content_type`**, and **`block_body`**.

## Route-level IP allowlist (per host)

Use **`rules[].waf.allow`** to restrict one hostname to internal or specific source IPs. It reuses the WAF IP phase — there is no separate IP-allowlist module.

```yaml
rules:
  - host: admin.example.com
    backend:
      service:
        name: admin-console
        port: 8080
    waf:
      enabled: true            # required: the route-level WAF must be on
      disable_builtin: true    # IP-only mode (do not run attack signatures)
      allow:
        - 10.0.0.0/8
        - 192.168.0.0/16
        - 203.0.113.7/32
      # deny:                  # optional; wins over allow
      #   - 203.0.113.66/32
```

Semantics:

| Configuration | Behaviour |
|---|---|
| no `waf`, or `allow` and `deny` both empty | unrestricted (**default**, existing behaviour) |
| `allow` non-empty | only sources matching `allow` pass; everything else gets **403** |
| `deny` non-empty | matching sources are always blocked (even if also in `allow`) |
| `allow` empty + `deny` non-empty | deny-only blacklist (advanced) |
| `allow`/`deny` set but the route WAF is not enabled | `ingress validate` fails with an explicit error — the allowlist is never a silent no-op |

Client IP resolution (security-critical):

- **Default**: only the real TCP peer is used. Ingress captures it before zoox's `realip` middleware rewrites `RemoteAddr` from `X-Forwarded-For` / `X-Real-IP` / `True-Client-IP`, so a client cannot spoof its source.
- **`waf.trusted_proxies`** (global): list your load balancer / ingress-controller CIDRs. Only a request whose TCP peer is inside those networks may use `X-Forwarded-For`; the client is the rightmost XFF hop that is not a trusted proxy.
- **`trust_proxy` + `xff_index`** (legacy): kept for backward compatibility. Prefer `trusted_proxies` for new deployments. `0.0.0.0/0` and `::/0` in `trusted_proxies` are rejected at validation time.

Blocking and observability:

- Blocked requests receive **403** and reuse the configured [`error_pages`](./configuration.md) 403 page.
- Access log: `waf_block=1`.
- WAF log: `[waf block] phase=ip allow` (or `phase=ip deny`) with the resolved `client_ip`.
- WAF events: `rule=ip allow` / `rule=ip deny` (visible in the Admin Console).

Admin Console:

- **Route editor → IP 白名单**: allow/deny lists, trusted-proxy inheritance, 403 response, and a built-in **trial** (`RemoteAddr` + `X-Forwarded-For` → allow/deny with reason).
- **WAF module**: `trusted_proxies` editor with the rightmost-untrusted algorithm documented.
- Route list and route detail show the allowlist state (`allow n` / `未限制`).

Runnable files: [`examples/ip-allowlist/`](https://github.com/go-zoox/ingress/tree/master/examples/ip-allowlist).

## Built-in starter rules

When **`disable_builtin`** is not `true`, the following rules are appended **before** custom entries in `waf.rules`. IDs are stable so you can **replace** a builtin by adding your own rule with the **same `id`** (see [`examples/waf/rule-merge-by-id.yaml`](https://github.com/go-zoox/ingress/tree/master/examples/waf/rule-merge-by-id.yaml)). Patterns use [Go `regexp`](https://pkg.go.dev/regexp/syntax) syntax.

| ID | Targets | Description |
|----|---------|-------------|
| `builtin:sqli-common` | `uri` | Common SQL-injection style probes in path + raw query (`union select`, `sleep(`, `benchmark(`, `; drop/truncate/alter table`, …). |
| `builtin:path-traversal` | `path` | Path traversal probes (`../`, `..\`, encoded `..`, `etc/passwd`). |
| `builtin:xss-lite` | `uri` | Light reflected-scripting probes (`<script`, `javascript:`, `on*=`-style event handlers). |
| `builtin:rce-probes` | `uri` | Command injection probes (pipes, shell commands). |
| `builtin:jndi-lookup` | `uri`, `headers` | JNDI-style `${…jndi:` injection. |
| `builtin:sensitive-files` | `path` | Sensitive paths (`.env`, `.git/`, admin consoles, …). |
| `builtin:ssrf-probes` | `uri` | Cloud metadata / `file://` / `gopher://` SSRF probes (excludes localhost to reduce OAuth false positives). |
| `builtin:scanner-ua` | `header:User-Agent` | Known scanner User-Agents. |
| `builtin:crlf-injection` | `uri`, `headers` | CRLF / response-splitting probes. |
| `builtin:php-ssti` | `uri` | PHP eval, `php://`, template-style probes. |

Per-rule enablement:

- **`disable_builtin: false`** (default): all built-in rules are on unless listed in **`waf.builtin_rules`** with `false`.
- **`disable_builtin: true`**: all built-in rules are off unless **`waf.builtin_rules`** sets a rule to `true`.
- Custom **`waf.rules[]`** entries support **`enabled: false`** (default is on when omitted).

Per built-in **action** (without disabling the rule):

```yaml
waf:
  builtin_rule_actions:
    builtin:scanner-ua: audit
    builtin:xss-lite: pass
```

Example:

```yaml
waf:
  enabled: true
  disable_builtin: false
  builtin_rules:
    builtin:scanner-ua: false
  rules:
    - id: block-secret-path
      enabled: false
      type: contains
      pattern: /internal
      targets: [path]
    - id: allow-health
      action: pass
      type: contains
      pattern: /healthz
      targets: [path]
```

Exact patterns (from [`core/waf/builtin.go`](https://github.com/go-zoox/ingress/blob/master/core/waf/builtin.go); change there if you fork):

```
builtin:sqli-common
(?is)(union\s+select\b|sleep\s*\(|benchmark\s*\(|;\s*(drop|truncate|alter)\s+table\b)

builtin:path-traversal
(?:\.\./|\.\.\\|%2e%2e%2f|%2e%2e\\\\|etc/passwd\b)

builtin:xss-lite
(?is)(<\s*script\b|javascript:\s*[a-z]|\bon(?:click|load|error|focus|blur|change|submit|mouse\w*|key\w*|touch\w*|pointer\w*|scroll|dblclick|drag\w*|drop|input|reset|select|wheel|copy|cut|paste|abort|contextmenu|message|unload|beforeunload)\s*=)
```

Starters can false-positive on unusual but legitimate traffic — use **`log_only`** or **`disable_builtin: true`** and replace with stricter custom rules as needed. Regression cases for common benign traffic live in **`core/waf/builtin_false_positive_test.go`**.

## Custom rules (`waf.rules`)

| Field | Notes |
|-------|-------|
| `id` | Required. Route-level `waf.rules` with the same `id` **replace** the global rule of that id. |
| `enabled` | Optional. Default **true**; set `false` to disable a custom rule without deleting it. |
| `action` | `block` (default), `audit` (log only, keep checking), or `pass` (allow on match, stop further signatures). |
| `log_only` | Legacy; when `action` is omitted, `true` means `audit`. |
| `type` | `regex` (default) or `contains` |
| `pattern` | Compiled at startup for `regex`. |
| `targets` | One or more of `path`, `query`, `uri`, `headers`, `header:…` |
| `allow_hosts` | Optional. Matching Host **skips this rule only** (same syntax as global `allow_hosts`). Use with same `id` as a builtin to scope exceptions without replacing the pattern. |

Runnable files: [`examples/waf/`](https://github.com/go-zoox/ingress/tree/master/examples/waf).

## See also

- [Configuration reference](./configuration.md)
- [WAF examples](../examples/waf.md)
