---
title: "feat: POST response cache with JSON body key fields"
status: active
created: 2026-05-28
type: feat
delivery: "M1–M2 shipped; M3 admin UI + docs shipped"
---

# feat: POST response cache with JSON body key fields

## Problem

Many internal APIs use **POST + JSON** for read-only queries (product detail, list filters, RPC-style `/api/...`). Bodies carry noisy fields (`timestamp`, `requestId`, extra nulls), so **full-body hashing** rarely hits. Backends are unlikely to be refactored to GET soon.

Ingress already supports `backend.cache` with **`cache.paths`** (first match wins) for GET/HEAD. We need **whitelist-only POST caching** on specific paths, with cache keys derived from **selected JSON fields** (e.g. `product.id`), not the entire body.

## Goals

| ID | Goal |
|----|------|
| G1 | Only APIs listed in `cache.paths` may use POST caching; all other POST requests bypass HTTP cache |
| G2 | Per path rule: optional `methods: [POST]` and `key_json: [dot paths]` for fingerprint |
| G3 | Runtime: missing field, non-JSON, or oversize body → **no cache read/write** (pass through to origin) |
| G4 | Backward compatible: existing configs unchanged (GET/HEAD only, no JSON key) |

## Non-goals (v1)

- JSONPath (`$.foo`) — dot segments only
- Array index syntax (`items[0].id`) — defer to v2
- `key_json_normalize` (prune body to subset then hash) — choice 3 in ideation
- Custom key scripts / templates
- Caching non-JSON POST (`application/x-www-form-urlencoded`, multipart)
- Admin UI fields (can follow in a separate PR)
- Bump `httpcache:v1` only when canonical format ships (see **Cache key version**)

## Requirements traceability

| Req | Mechanism |
|-----|-----------|
| R1 | `cache.default: bypass` + explicit `paths[].action: cache` |
| R2 | `paths[].methods` overrides backend-level `cache.methods` for that rule |
| R3 | `paths[].key_json` list of dot paths |
| R4 | `paths[].key_body_max_bytes` caps bytes read for JSON parse + fingerprint |
| R5 | `buildHTTPCacheCanonical` appends sorted `jsonkey:<path>=<scalar>` lines |
| R6 | Service proxy: buffer request body before cache lookup; replay on upstream |
| R7 | Store upstream response for allowed POST same as GET guards |

---

## Configuration schema

### `core/rule/backend_cache.go` — extend `BackendCachePathRule`

```go
type BackendCachePathRule struct {
	Match        string   `config:"match"`
	MatchType    string   `config:"match_type"`
	Action       string   `config:"action"`
	TTL          int64    `config:"ttl"`
	MaxBodyBytes int64    `config:"max_body_bytes"`

	// Methods overrides backend.cache.methods for this path rule when non-empty.
	// Example: [POST] for JSON query APIs. When empty, inherit backend.cache.methods (default GET, HEAD).
	Methods []string `config:"methods"`

	// KeyJSON lists dot-separated paths into the request JSON object for cache fingerprint.
	// Example: ["product.id", "storeId"]. Required when this rule allows POST and action is cache.
	KeyJSON []string `config:"key_json"`

	// KeyBodyMaxBytes limits request body size read for JSON parsing (default 65536 when key_json set).
	// Bodies larger than this skip cache for that request. 0 means use default at compile time.
	KeyBodyMaxBytes int64 `config:"key_body_max_bytes"`
}
```

### `BackendCachePathRuleCompiled` — compile-time copies

```go
type BackendCachePathRuleCompiled struct {
	// ... existing fields ...
	Methods         []string // uppercased, non-empty when set on rule
	KeyJSON         []string // validated dot paths
	KeyBodyMaxBytes int64    // resolved default 65536 if key_json non-empty and yaml 0
}
```

### Global `BackendCache` — no new top-level fields (v1)

- Keep `cache.methods` as backend default (GET, HEAD).
- POST + JSON keys live **only** on `paths[]` entries that need them.

### Recommended operator pattern

```yaml
cache:
  enabled: true
  default: bypass          # mandatory for POST whitelist setups
  ttl: 300
  paths:
    - match: /api/product/getDetail
      match_type: exact
      action: cache
      methods: [POST]
      key_json:
        - product.id
        - lang
      ttl: 120
      key_body_max_bytes: 32768
      max_body_bytes: 2097152   # max upstream *response* body to store
```

---

## Runtime semantics

### Request flow (service backend)

```
match route → rate limit → WAF
  → resolve httpCachePolicyForRequest(path)  # merges path rule methods/key_json
  → if policy allows method:
       if POST (or rule requires key_json):
         read body ≤ key_body_max_bytes into buffer (clone)
         parse JSON; extract key_json fields
         on failure → skip cache (no lookup, no store), forward with buffered body
       build storage key (includes jsonkey lines)
       try cache hit → return
  → proxy upstream (body = buffer)
  → OnResponse: if may store && method allowed → Set cache
```

### Request body replay (does not break proxy)

`http.Request.Body` is a one-shot `io.ReadCloser`. Reading it for JSON key extraction **consumes** the stream unless we replace it.

**Required behavior (M2):**

1. Read at most `key_body_max_bytes` into a **new byte slice** (`append([]byte(nil), buf...)` or `bytes.Clone`) — never retain a reference to the underlying read buffer if reused.
2. Replace `ctx.Request.Body` with `io.NopCloser(bytes.NewReader(cloned))`.
3. Set `ctx.Request.ContentLength = int64(len(cloned))` and `ctx.Request.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(cloned)), nil }` so the outbound proxy / `RoundTripper` can re-read the same payload (retries, HTTP/2, go-zoox proxy).
4. If the client sent `Content-Length` or `Transfer-Encoding: chunked`, leave header handling to the proxy after body replacement; do not forward with an empty body.

Cache **hit** short-circuits before upstream: body clone is only for keying on that path; no upstream round-trip.

Cache **skip** (parse fail, oversize, non-JSON): still **must** replay the cloned body to upstream when the request continues to the service backend.

This mirrors the existing **response** path in `core/build.go` (`io.ReadAll(res.Body)` then `res.Body = io.NopCloser(bytes.NewReader(body))`), applied to the **request** side once per request.

### Content-Type gate

Treat as JSON when `Content-Type` is empty **or** parses as `application/json` (allow `application/json; charset=utf-8`). Otherwise: **skip cache** for that request.

### Dot-path extraction (`product.id`)

1. Decode body with `json.Decoder` / `Unmarshal` into `map[string]any` (top-level object only; reject top-level array).
2. Walk segments split on `.` (no escapes in v1).
3. At each step, value must be `map[string]any` until the last segment.
4. **Leaf types allowed**: `string`, `float64` (JSON number), `bool`, `json.Number`.
5. **Rejected leaf types** (skip cache): `nil`, `map`, `slice`, nested non-scalar.
6. Canonical scalar encoding:
   - string: as-is (UTF-8)
   - bool: `true` / `false`
   - number: `strconv.FormatFloat` with `'f', -1, 64` or `json.Number.String()`

### Canonical fingerprint extension

After existing lines (`method`, `scheme`, `host`, `path`, `query`, hashed `key_headers`), append **sorted by path**:

```
jsonkey:lang=zh-CN
jsonkey:product.id=8848
```

Then hash with existing `key_hash` (md5/sha256). Prefix remains `httpcache:v1:` until we intentionally version (see below).

### Method inheritance

| `paths[i].methods` | Effective methods for matched path |
|--------------------|------------------------------------|
| empty | `backend.cache.methods` (default GET, HEAD) |
| `[POST]` | POST only for that path |
| `[GET, POST]` | both (unusual; allowed) |

When matched path rule has `key_json` non-empty, **`POST` must appear** in effective methods (validate at config load).

### Store (write) rules

- Same eligibility as today: status/body size/`Set-Cookie`/`Cache-Control` via `httpCacheShouldStore`.
- Extend `build.go` store guard: allow store when `httpCacheMethodAllowed(method, pc)` includes POST (not only `MethodGet`).
- **Do not** store if JSON key extraction failed on the request (even if upstream returned 200).

### Cache key version

**Decision:** Use prefix **`httpcache:v2:`** whenever a matched path policy uses `key_json` (non-empty). Global GET-only caches without any `key_json` in config stay on **`httpcache:v1:`**.

---

## Validation rules (`validateBackendCache` + `compileBackendCachePathRules`)

| Condition | Error |
|-----------|-------|
| `paths[i].key_json` set, `action` is `bypass` | `{loc}: backend.cache.paths[i].key_json requires action cache` |
| `paths[i].key_json` non-empty, effective methods lack POST | `{loc}: backend.cache.paths[i].key_json requires POST in paths[i].methods or backend.cache.methods` |
| `paths[i].methods` contains only POST, `key_json` empty | `{loc}: backend.cache.paths[i].methods includes POST but key_json is empty (required for POST cache)` |
| `paths[i].key_json[j]` empty or whitespace | `{loc}: backend.cache.paths[i].key_json[j] must be non-empty` |
| `paths[i].key_json[j]` invalid syntax | `{loc}: backend.cache.paths[i].key_json[j] must be dot-separated identifiers (e.g. product.id)` |
| Invalid identifier segment | `{loc}: backend.cache.paths[i].key_json[j]: segment "0foo" must match [A-Za-z_][A-Za-z0-9_]*` |
| `paths[i].key_body_max_bytes` < 0 | `{loc}: backend.cache.paths[i].key_body_max_bytes must be >= 0` |
| `paths[i].key_body_max_bytes` > 1MiB (suggested cap) | `{loc}: backend.cache.paths[i].key_body_max_bytes must be <= 1048576` |
| Global `cache.methods` contains POST (any entry) | `{loc}: backend.cache.methods must not include POST; use cache.paths[].methods and key_json` |

**Dot-path syntax (compile time):** segments match `^[A-Za-z_][A-Za-z0-9_]*$`; at least one segment; max 32 segments; max 256 chars total per path string.

**Compile errors** (reuse `loc` prefix):

| Condition | Error |
|-----------|-------|
| regex compile fails | `{loc}: backend.cache.paths[i].match regex: ...` (existing) |

---

## Runtime skip reasons (debug logs, optional metric)

Log at debug when cache skipped for POST rule:

| Reason | Message fragment |
|--------|------------------|
| Body too large | `http cache skip: request body exceeds key_body_max_bytes` |
| Not JSON | `http cache skip: content-type not json` |
| Parse error | `http cache skip: json parse failed` |
| Missing path | `http cache skip: key_json path not found: product.id` |
| Non-scalar leaf | `http cache skip: key_json non-scalar: filters` |

Access log: no change required v1; optional future `cache_skip_reason=` field.

---

## Examples (illustrative bad backends)

### 1. Product detail POST

```yaml
# POST /api/product/getDetail  body: { "product": { "id": 123 }, "timestamp": 1716..., "trace": "..." }
- match: /api/product/getDetail
  match_type: exact
  action: cache
  methods: [POST]
  key_json:
    - product.id
    - lang
  ttl: 300
```

Same `product.id` + `lang` hits cache regardless of `timestamp` / `trace`.

### 2. List / search with filter object (scalar only)

```yaml
# Backend returns list for POST /api/goods/search with body.filters.categoryId + page
- match: /api/goods/search
  match_type: exact
  action: cache
  methods: [POST]
  key_json:
    - filters.categoryId
    - page
    - pageSize
  ttl: 60
  key_body_max_bytes: 16384
```

If `filters` is an object and `categoryId` is inside it, path `filters.categoryId` works. If dev nests unparsed arrays, extraction fails → bypass (forces them to fix or add fields).

### 3. Prefix whitelist for a family of RPC paths

```yaml
cache:
  enabled: true
  default: bypass
  paths:
    - match: /legacy/rpc/query/
      match_type: prefix
      action: cache
      methods: [POST]
      key_json:
        - service
        - method
        - params.bizId
      ttl: 30
```

Covers multiple URLs under prefix sharing one key shape (common in legacy gateways).

### 4. Explicit bypass for dangerous POST under same prefix

```yaml
paths:
  - match: /legacy/rpc/query/submitOrder
    match_type: exact
    action: bypass
  - match: /legacy/rpc/query/
    match_type: prefix
    action: cache
    methods: [POST]
    key_json: [service, method, params.orderId]
```

First match wins — submit order never cached.

---

## Tests (implementation checklist)

| Test | Assert |
|------|--------|
| `validate` POST without `key_json` | error |
| `validate` `key_json` with GET-only methods | error |
| `validate` invalid `key_json` segment `product..id` | error |
| `buildHTTPCacheCanonical` with json fields | stable order, HEAD still folds to GET for key |
| extract `product.id` | hit across bodies differing only in noise fields |
| missing `product.id` | no store, no hit |
| body > `key_body_max_bytes` | skip |
| `Content-Type: text/plain` + POST rule | skip |
| store POST 200 response | entry retrievable |
| `default: bypass` + unmatched POST path | no cache |

---

## Implementation touchpoints

| File | Change |
|------|--------|
| `core/rule/backend_cache.go` | New fields on path rule + compiled |
| `core/http_cache.go` | `httpCacheRuntime` per-request policy copy with Methods/KeyJSON; canonical builder; JSON extract helper; optional v2 prefix |
| `core/http_cache.go` | `httpCachePathDecision` merge methods + key fields into effective policy |
| `core/build.go` | Buffer body; POST store; replay body on proxy |
| `core/validate.go` | Rules table above |
| `examples/advanced/http-response-cache-post-json.yaml` | Runnable sample |
| `docs/guide/caching.md` + `docs/zh/guide/caching.md` | Operator guide |
| `AGENTS.md` | Short note on POST/jsonkey |

---

## Decisions (confirmed)

| # | Decision |
|---|----------|
| 1 | **`httpcache:v2:`** when `key_json` is used on a path rule; otherwise v1 for legacy GET caches |
| 2 | **Forbid POST** in top-level `backend.cache.methods`; POST only via `cache.paths[].methods` + `key_json` |
| 3 | **Empty object `{}`** with required `key_json` → all paths missing → skip cache (no read/write) |

---

## Delivery phases

| Phase | Scope |
|-------|--------|
| M1 | Schema + validate + compile + canonical/json extract unit tests |
| M2 | `build.go` integration + example + docs |
| M3 | Admin form fields for `methods`, `key_json`, `key_body_max_bytes` on path rows |
