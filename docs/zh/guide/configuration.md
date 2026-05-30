# 配置参考

Ingress 使用 YAML 配置文件来定义路由规则、认证、SSL 证书和其他设置。

## 配置结构

```yaml
port: 8080                     # HTTP 端口（默认：8080）
# enable_h2c: false            # 可选：在 HTTP 端口启用明文 HTTP/2（h2c）；公网不建议开启

# 缓存配置
cache:
  ttl: 30                      # 缓存 TTL（秒）
  # engine: redis              # 可选：使用 Redis 缓存
  # host: 127.0.0.1
  # port: 6379
  # password: '123456'
  # db: 2

# HTTPS 配置
https:
  port: 8443                   # HTTPS 端口
  # enable_http3: false        # 可选：启用 HTTP/3（QUIC，UDP）；需已配置 TLS
  # http3_port: 8443           # 可选：UDP 端口（默认与 https.port 相同）
  # http3_altsvc_max_age: 86400 # 可选：Alt-Svc 的 ma=（秒）；负数表示不发送该头
  # redirect_from_http:
  #   enabled: true            # 可选：默认 false；设为 true 以在已配置 https.port 时强制 HTTP -> HTTPS
  #   permanent: true          # 可选：true=301，false=302
  #   exclude_paths:           # 可选：跳过重定向的精确路径
  #     - /healthz
  ssl:
    - domain: example.com
      cert:
        certificate: /path/to/cert.pem
        certificate_key: /path/to/key.pem

# 健康检查配置
healthcheck:
  outer:
    enable: true               # 启用外部健康检查
    path: /healthz             # 健康检查端点路径
    ok: true                   # 始终返回 OK
  inner:
    enable: true               # 启用内部服务健康检查
    interval: 30               # 检查间隔（秒）
    timeout: 5                 # 检查超时（秒）

# 无规则匹配时使用的 fallback
fallback:
  service:
    protocol: https
    name: httpbin.org

# 路由规则
rules:
  - host: example.com
    backend:
      # 仅配置一种后端形态时可省略 backend.type — 对照示例：examples/basic/ingress.yaml（显式 type: service / 省略）
      service:
        name: backend-service
        port: 8080
```

## 配置字段

### 顶级字段

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `port` | int | 监听的 HTTP 端口 | `8080` |
| `enable_h2c` | bool | 在 HTTP 端口启用明文 HTTP/2（h2c） | `false` |
| `cache` | object | 应用层 `ctx.Cache()`（内存或 Redis）；承载匹配器数据及可选的 **`backend.cache`** 条目 | - |
| `https` | object | HTTPS 配置 | - |
| `healthcheck` | object | 健康检查配置 | - |
| `fallback` | object | 回退后端 | - |
| `rules` | array | 路由规则 | `[]` |
| `waf` | object | WAF 基线；路由级补丁为 **`rules[].waf`** 映射（参见 [WAF](waf.md)） | 省略或 `enabled: false` 时不启用 |
| `security` | object | 安全响应头预设（HSTS / frame / CSP / CORS）；路由级 **`rules[].security`** | 省略或 `profile: off` 时不启用 |
| `maintenance` | object | 全局维护域名列表与默认 503 设置（参见 [维护模式](maintenance.md)） | 省略时不启用 |
| `logging` | object | Zoox 日志配置（控制台 + 可选文件 transport）；见 [Logging](#logging-日志) | 省略时仅控制台 |
| `admin` | object | 内嵌运维控制台（参见 [Admin 指南](admin.md)） | 省略时不启用 |

### 维护（`maintenance` / `rules[].backend.service.maintenance`）

在路由匹配与 WAF 之后判定；于重定向 / Handler / 上游之前返回 **503**。详见 [维护模式](maintenance.md)。

**全局 `maintenance:`**

| 字段 | 类型 | 说明 |
|------|------|------|
| `hosts` | array | 域名条目，格式为 `{ host, window? }` 对象；可设 `window.start` / `window.end`（RFC3339） |
| `retry_after` | int | `Retry-After` 响应头（秒，`0` 表示不发送） |
| `title` / `subtitle` | string | 503 页面标题 / 说明 |
| `bypass.allow_ips` | string 数组 | 客户端 IP/CIDR 白名单 |
| `bypass.paths` | string 数组 | 精确路径或后缀 `*` 前缀匹配 |
| `bypass.header.name` / `value` | string | 请求头放行键值对 |

**内置状态探测：** `GET /_/ingress/status` — 按请求 Host 返回 JSON `{"status":"ok"}`（200）或 `{"status":"maintenance",...}`（503）；详见 [维护模式](maintenance.md#ingress-状态探测)。不可配置。

**路由 `rules[].backend.service.maintenance`**（仅 Host 级 **service** 后端）：

| 字段 | 类型 | 说明 | 默认 |
|------|------|------|------|
| `enabled` | bool | 启用路由维护 | `false` |
| `scope` | string | `all` 或 `listed` | `all` |
| `hosts` | array | `scope: listed` 时必填；格式同全局 `hosts` | — |
| `retry_after` | int | 路由维护命中时覆盖全局 | `0` |
| `title` / `subtitle` | string | 路由维护命中时覆盖全局 | — |
| `bypass` | object | 与全局 bypass 合并 | — |

维护 503 的访问日志附加 `maintenance_block=1`；维护 503 响应包含 **`X-Ingress-Maintenance: true`**（上游 503 不会）。

### WAF（`waf` / `rules[].waf`）

| 字段 | 类型 | 描述 |
|------|------|------|
| `enabled` | bool | 总开关 |
| `trust_proxy` | bool | 是否从 `X-Forwarded-For` 解析客户端 IP |
| `xff_index` | int | 选第几段（`0`=最左；负数从右数） |
| `log_only` | bool | 全局仅审计不打断请求 |
| `block_status_code` | int | 拦截时 HTTP 状态码（默认 403，`0` 表示默认） |
| `block_content_type` | string | 拦截响应 `Content-Type` |
| `block_body` | string | 拦截响应体 |
| `disable_builtin` | bool | `true` 时关闭内置 starter 规则（清单见 [WAF](waf.md)） |
| `deny` | string 数组 | 拒绝的 IP/CIDR（先匹配） |
| `allow` | string 数组 | 非空时仅允许表中网段通过 IP 阶段 |
| `allow_hosts` | string 数组 | 域名白名单：匹配的 Host 跳过全部 WAF（精确、`*` 通配或 Go 正则，推断规则同路由 host） |
| `rules` | array | 自定义特征（`id`、`pattern`、`type`、`targets`、可选 `allow_hosts`、`log_only`、`action`）；同 `id` 可覆盖内置或路由级继承字段 |

### 安全响应头（`security` / `rules[].security`）

按 **profile** 自动添加 HSTS、X-Frame-Options、CSP、CORS 等响应头；详见 [安全响应头](security-headers.md)。

| 字段 | 类型 | 描述 |
|------|------|------|
| `profile` | string | `strict` / `api` / `embeddable` / `off` |
| `hsts` | string | `auto`（仅 HTTPS）、`on`、`off` |
| `frame` | string | `inherit` / `deny` / `sameorigin` / `off` |
| `cors.origins` | 数组 | 允许的 Origin（`api` 预设必填） |

`api` 预设启用 CORS；Ingress 会直接响应 OPTIONS 预检。

### 缓存配置

顶层 **`cache`** 配置共享的 Zoox **`ctx.Cache()`** 后端（内存或 Redis）：用于**匹配器 / 路由**等数据；若某条 **`backend.cache`** 开启，**HTTP 响应**条目也写入同一后端（详见下文 [`backend.cache`](#backendcache-http-响应缓存) 与[缓存指南](caching.md)）。

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `engine` | string | `memory` 或 `redis` | `memory` |
| `ttl` | int | 默认 TTL（秒），用于匹配键等 | `60` |
| `host` | string | Redis 主机（如果使用 Redis） | - |
| `port` | int | Redis 端口 | `6379` |
| `password` | string | Redis 密码 | - |
| `db` | int | Redis 数据库编号 | `0` |
| `prefix` | string | `ctx.Cache()` 中所有键的前缀（匹配器数据与 `httpcache:v1:` HTTP 缓存条目都会带上此前缀） | - |

### HTTPS 配置

| 字段 | 类型 | 描述 |
|------|------|------|
| `port` | int | 监听的 HTTPS 端口 |
| `enable_http3` | bool | 在已配置 TLS 时启用 HTTP/3（QUIC，UDP） |
| `http3_port` | int | HTTP/3 的 UDP 端口；`0` 表示与 `https.port` 相同 |
| `http3_altsvc_max_age` | int | `Alt-Svc` 的 `ma=`（秒）；`0` 使用服务端默认；负数为不发送 `Alt-Svc` |
| `redirect_from_http.enabled` | bool | 启用全局 HTTP -> HTTPS 强制重定向（默认 `false`，设为 `true` 以在设置 `https.port` 时激活） |
| `redirect_from_http.permanent` | bool | 为 `true` 时使用 `301`，否则使用 `302` |
| `redirect_from_http.with_origin_method_and_body` | bool | 为 `true` 时使用 `308`/`307` 以保留方法与请求体（默认 `false`，否则为 `301`/`302`） |
| `redirect_from_http.exclude_paths` | array | 跳过强制重定向的精确路径 |
| `ssl` | array | SSL 证书配置 |

启用 HTTPS 后，TLS 上的 HTTP/2 由运行时自动协商（ALPN），无需单独配置项。

#### SSL 证书

| 字段 | 类型 | 描述 |
|------|------|------|
| `domain` | string | 证书的域名 |
| `cert.certificate` | string | 证书文件路径 |
| `cert.certificate_key` | string | 私钥文件路径 |

### 健康检查配置

#### 外部健康检查

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enable` | bool | 启用外部健康检查 | `false` |
| `path` | string | 健康检查端点路径 | `/healthz` |
| `ok` | bool | 始终返回 OK | `false` |

#### 内部健康检查

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enable` | bool | 启用内部健康检查 | `false` |
| `interval` | int | 检查间隔（秒） | `30` |
| `timeout` | int | 检查超时（秒） | `5` |

### 回退配置

当没有路由规则匹配请求时，使用回退后端。

若 **未** 设置 `service.request.host.rewrite`，发往 fallback 的 `Host` 仍会对齐到该回退上游。若需保留客户端 `Host`，请显式设置 `service.request.host.rewrite: false`。也可在 **`service.mode: external`** 表达「对齐上游 Host」（与省略 `rewrite` 时一致）。

```yaml
fallback:
  service:
    # mode: internal            # 可选 — 推荐写在 service 下；internal（默认）| external
    name: fallback-service
    port: 8080
    # protocol: 可选，默认 http
    protocol: http
    # request:
    #   host:
    #     rewrite: false        # 可选：保留客户端 Host
```

### 规则配置

规则定义如何将请求路由到后端服务。详细信息请参阅[路由指南](/zh/guide/routing)。

每个 **`backend.service`**：**省略 `protocol` 时默认为 `http`**（`core/service/service.go` 的配置默认值，与 `core/service/host.go` 一致）。**`protocol: https`** 且省略 **`port`**（或为 `0`）时上联端口默认为 **443**；**`http`**（显式或默认）时省略 **`port`** 默认为 **80**。影响出站 URL 与默认 **`Host`** 头。

**`service.mode`**（`internal` 默认，`external` 用于第三方源站）在省略 **`request.host.rewrite`** 时控制发往上游的 **`Host`**，见[重写](rewriting.md)。**`backend.mode`** 仍可用，但须与 **`service.mode`** 一致。

下面 **`backend.type` 写法混排**：规则级 `backend` **显式写 `type: service`**；各 **`paths[].backend`** **省略 `type`**，由各自的配置块推断 **`service`** 或 **`handler`**。

```yaml
rules:
  - host: example.com           # 要匹配的主机
    # host_type: 可选 — 省略或写 auto 时在编译阶段根据 host 推断 exact / regex / wildcard
    # 显式取值：exact、regex、wildcard
    backend:
      type: service             # 可选 — 仅配 service 时可省略（见 examples/basic/ingress.yaml）
      service:
        name: backend-service
        port: 8080
        # mode: internal        # 可选 — internal（默认）| external；优先写在 service 下
        # protocol: 可选，默认 http；上联为 TLS 时写 https
        protocol: http
        auth:                   # 认证（可选）
          type: basic
          basic:
            users:
              - username: admin
                password: admin123
        healthcheck:            # 服务健康检查（可选）
          enable: true
          method: GET
          path: /health
          status: [200]
        request:
          host:
            rewrite: true       # 可选显式覆盖；若已设 service.mode: external 常可省略
          path:
            rewrites:          # 路径重写规则
              - ^/api/v1:/api/v2
          headers:             # 附加头
            X-Custom-Header: value
          query:               # 查询参数
            key: value
          delay: 0              # 延迟（毫秒）
          timeout: 30           # 超时（秒）
      # redirect: ...           # 仅 redirect 块 — 详见路由指南（唯一时可省略 backend.type）
    paths:                      # 基于路径的路由（可选）
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
            # mode: internal      # 路径级可选 — 写在 service 下
      - path: /healthz
        backend:
          handler:
            status_code: 200
            headers:
              Content-Type: application/json
            body: |
              {"ok": true}
```

### `rules[].backend` 与 `paths[].backend` 字段

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `type` | string | `service`、`handler` 或 `redirect`（常**省略**由配置块推断） | 推断 |
| `mode` | string | 兼容字段：与 **`backend.service.mode`** 相同语义；优先读 **`service.mode`**；两者非空时必须一致 | `internal` |
| `service` | object | `service` 型时的上联配置 | - |
| `handler` | object | `handler` 型 | - |
| `redirect` | object | `redirect` 型 | - |
| `cache` | object | 可选 HTTP 响应缓存，适用于 **service** / **handler** / **redirect**；见下文 | 关闭 |

**Host 上游** 的 **`internal` / `external`** 以 **`backend.service.mode`** 为准（若未设置则回退 **`backend.mode`**）；两者不能设为不同值。**handler** / **redirect** 不得写 **`service.mode`**。

#### `backend.cache`（HTTP 响应缓存）

- **适用于** `backend.type: service`、`handler`、`redirect`（显式或推断）。**默认关闭**，除非 `cache.enabled: true`。
- **存储**与路由匹配器共用 Zoox 应用缓存（`ctx.Cache()`）：由顶层 `cache` 配置 Redis 或内存（`core/prepare.go`）。键前缀为 **`httpcache:v1:`**（路径规则配置了 **`key_json`** 时为 **`httpcache:v2:`**），指纹为规范请求串的 MD5 或 SHA-256（方法、scheme、host、path、排序后的 query、参与键的请求头——头值经哈希，以及可选的 **`jsonkey:`** 行）。
- **HEAD** 与 **GET** 共用同一缓存键；**GET**（及路径允许的 **POST**）可写入缓存：**反代**在 `OnResponse` 落盘；**handler** 在配置允许时捕获 body；**redirect** 在 URL 展开后写入 `Location` 与状态码（redirect 写入仍仅 **GET**；避免空 HEAD 覆盖完整 GET）。
- **客户端绕过**（不读不写缓存）：请求 `Cache-Control` 含 `no-cache`、`no-store` 或 `max-age=0`（可配置）、在默认开启 `honor_pragma_no_cache` 时含 `Pragma: no-cache`、或请求带 **`Range`**。
- **不写入**（service / handler 带 body）：非 200；**非空 `Vary`** 默认阻止落盘（可按 `Vary` 拆键未实现，见[缓存指南](caching.md)）；若 **`cache.skip_vary: true`**，则**不保存、不下发** `Vary`（仍以单一变体对外，需自担语义风险）；`no-store`；`private`（除非 `ignore_response_private: true`）；**响应含 `Set-Cookie`** 且 `skip_when_set_cookie` 为 **true**（默认）时；body 大于 `max_body_bytes`。**redirect** 可缓存 301/302/303/307/308 及 `Location`（无 body；适用相同的 Cache-Control / `Set-Cookie` 等规则）。许多 httpbin 路径带 `Vary: Origin`；需共享缓存时可设 **`skip_vary: true`**（仅限你可接受「一剂到底」的场景）。
- **验证命中**：对同一 URL 连续发两次不带绕过条件的 **GET**；第二次应走缓存。命中时访问日志行尾附加 **`cache_hit=1`**（含 handler、redirect 分支）。

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `enabled` | bool | 为本 backend 打开 HTTP 响应缓存 | `false` |
| `ttl` | int | 上游未给出更严格的 `max-age` / `s-maxage` 时的最长新鲜时间（**秒**） | `300` |
| `max_body_bytes` | int | 大于此大小的响应体不缓存（代码中 ≤0 或未设置时默认 **2MiB**） | `2097152` |
| `key_hash` | string | 指纹算法：`md5` 或 `sha256` | `md5` |
| `methods` | string 数组 | 允许参与缓存的方法（运行时规范为大写）。**不得包含 `POST`**——POST 查询 API 请用 `paths[].methods` + `key_json`。 | `GET`、`HEAD` |
| `key_headers` | string 数组 | 参与缓存键的请求头名（头值为哈希摘要，不存原文）。名称经 `http.CanonicalHeaderKey` 规范化，**不区分大小写**去重。 | *（无默认；留空则请求头不参与键）* |
| `bypass_request_directives` | string 数组 | 命中则跳过缓存读写、按正常逻辑处理的 `Cache-Control` 记号 | `no-cache`、`no-store`、`max-age=0` |
| `honor_pragma_no_cache` | bool | 将 `Pragma: no-cache` 视为与 `Cache-Control: no-cache` 等同以绕过缓存 | `true` |
| `ignore_response_private` | bool | 是否允许缓存标为 `Cache-Control: private` 的响应 | `false` |
| `skip_when_set_cookie` | bool | 为 **true**（默认）时，**不缓存**含 **`Set-Cookie`** 的响应；仅在高阶场景下可设为 `false`（可能缓存到带会话的个性化内容，需谨慎）。 | `true` |
| `skip_vary` | bool | 为 **true** 时允许缓存带 **`Vary`** 的响应（**不写入**也**不返回** `Vary` 头）；仅当上游对该 URL 实际可视为单变体时使用 | `false` |
| `default` | string | 配置了 **`paths`** 时，未命中任何规则的路径：`cache` 或 `bypass` | `cache` |
| `paths` | array | 有序路径规则（**先匹配先生效**）；见下表 | — |

**`backend.cache.paths[]`**（可选）：

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `match` | string | 路径模式（必填） | — |
| `match_type` | string | `auto`、`prefix`、`exact` 或 `regex` | `auto` |
| `action` | string | `cache`（读写缓存）或 `bypass`（完全跳过缓存） | `cache` |
| `ttl` | int | `action: cache` 且 `> 0` 时覆盖 backend 的 `ttl` | 继承 |
| `max_body_bytes` | int | `action: cache` 且 `> 0` 时覆盖 backend 的 `max_body_bytes` | 继承 |
| `methods` | string 数组 | 非空时覆盖本路径的 `methods`（如 `[POST]`） | 继承 |
| `key_json` | string 数组 | 请求 JSON 的点分路径，参与指纹（如 `product.id`）；本规则 **`methods` 须含 `POST`**；使用 **`httpcache:v2:`** | — |
| `key_body_max_bytes` | int | 配置 `key_json` 时读取请求 body 的上限（`0` → 编译期默认 **65536**） | 有 `key_json` 时为 **65536** |

**`match_type: auto`**（与 `host_type: auto` 同类推断）：含 `( ) [ ] ^ $ | + ? \` → **regex**；以 `/` 结尾 → **prefix**；否则 **exact**。规则按列表顺序匹配；更窄的模式应写在更宽的前面（例如先 `bypass` `/static/private`，再 `cache` `/static/`）。**未配置 `paths`** 时，行为与原先一致：`enabled: true` 则该 backend 下所有路径参与缓存。

示例：[`examples/advanced/http-response-cache.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/http-response-cache.yaml)（内存 `ctx.Cache()`）、[`examples/advanced/redis-cache.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/redis-cache.yaml)（Redis + `backend.cache`）、[`examples/advanced/http-response-cache-paths.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/http-response-cache-paths.yaml)（按路径规则）、[`examples/advanced/http-response-cache-post-json.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/advanced/http-response-cache-post-json.yaml)（POST + `key_json`）。

实现见 `core/rule/backend_cache.go`、`core/http_cache.go`、`core/build.go`。

### Admin（`admin`）

可选内嵌控制台（HTTP API + UI）。**`admin.enabled: true`** 时与 **`ingress run`** 同进程启动。完整说明：[Admin 控制台](admin.md)。

| 字段 | 类型 | 描述 | 默认值 |
|------|------|------|--------|
| `admin.enabled` | bool | 与代理一起启动 admin | `false` |
| `admin.port` | int | Admin 监听端口 | `9080` |
| `admin.database.driver` | string | 审计 / 修订 SQLite 驱动 | `sqlite` |
| `admin.database.dsn` | string | 数据库 DSN | `file:admin.db?cache=shared&_fk=1` |
| `admin.web.dev_proxy` | bool | 仅 API；UI 由 Vite 开发服务器提供 | `false` |
| `admin.access_log_path` | string | 日志页 access 路径 | 来自 `logging` |
| `admin.error_log_path` | string | 日志页 error 路径 | 来自 `logging` |

```yaml
admin:
  enabled: true
  port: 9080
  database:
    driver: sqlite
    dsn: file:./admin.db?cache=shared&_fk=1
```

示例包：[`examples/admin-console/ingress.yaml`](https://github.com/go-zoox/ingress/blob/master/examples/admin-console/ingress.yaml)。

## Logging（`logging`）

`logging` 块对应 [zoox](https://github.com/go-zoox/zoox) `Config.Logger`（字段相同；ingress 在 prepare 时复制到 `app.Config.Logger`）。Zoox 始终包含**控制台**输出；`transports` 可叠加文件等 sink。

| 字段 | 类型 | 描述 |
|------|------|------|
| `logging.enable` | bool | 为 **true** 时启用控制台 + 文件日志。未写 `transports` 时默认 `/var/log/ingress/access.log` 与 `error.log`（目录自动创建）。为 **false** 时仅控制台。当 **`admin.enabled: true`** 且 **未配置 `logging`** 时，默认 **`enable: true`**，并在配置文件同目录使用 **`access.log`** / **`error.log`**。**显式 `logging.*` 始终优先。** |
| `logging.level` | string | 最低级别（`debug`、`info`、`warn`、`error`） |
| `logging.transports` | array | 额外 sink，如 `type: file` 与 `path`、`levels` |
| `logging.middleware.disabled` | bool | Ingress 设为 `true`（关闭 zoox HTTP 请求日志中间件） |

示例：

```yaml
logging:
  enable: true
  level: warn
```

自定义路径：

```yaml
logging:
  enable: true
  level: warn
  transports:
    - type: file
      path: /var/log/ingress/access.log
      levels:
        error: /var/log/ingress/error.log
```

省略 `logging` 时使用 zoox 默认（仅控制台）。

## 访问日志字段

Ingress 的访问日志为应用侧固定格式（非 Nginx `log_format` 配置项），在保留原有主字段的基础上追加以下扩展字段：

- `referer`：对应请求头 `Referer`，为空时为 `-`
- `ua`：对应请求头 `User-Agent`，为空时为 `-`
- `xff`：对应请求头 `X-Forwarded-For`，为空时为 `-`
- `real_ip`：优先取请求头 `X-Real-IP`，否则回退到请求连接地址，无法获取时为 `-`
- `tls_protocol`：TLS 协议版本（如 `TLS 1.3`），非 TLS 请求为 `-`
- `tls_cipher`：TLS cipher suite 名称，非 TLS 请求为 `-`
- `upstream_status`：上游响应状态码（handler 分支使用 handler 状态码）
- `upstream_response_length`：上游响应长度（未知时可能为 `-1`）
- `upstream_response_time`：上游响应耗时（Go `time.Duration` 文本格式）
- `cache_hit`：响应由 **`backend.cache`** 命中时附加 **`cache_hit=1`**（含 service 反代、handler、redirect）；未命中或未启用缓存的路由不会出现该字段

示例：

```text
[host: example.com, target: http://backend:8080] "GET /api HTTP/1.1" 200 12.3ms real_ip="10.0.0.9" referer="https://portal.example.com/" ua="curl/8.7.1" xff="10.0.0.1" tls_protocol="TLS 1.3" tls_cipher="TLS_AES_128_GCM_SHA256" upstream_status=200 upstream_response_length=512 upstream_response_time=12.3ms
```

说明：当前未提供与 Nginx `$body_bytes_sent` 完全等价的独立字段；如需该指标，建议通过下游日志平台从响应统计补充。

## 环境变量

您可以使用环境变量覆盖某些配置：

- `CONFIG`: 配置文件路径
- `PORT`: HTTP 监听端口；设置时会**覆盖** YAML 顶层 **`port`**

## 配置验证

Ingress 在进程启动以及执行 **`ingress validate`** 时会校验配置；校验失败将无法启动或无法成功 reload。

静态检查包括路由编译（host/path 正则）、HTTPS/证书结构、每条 backend 的 **`mode`（`internal` / `external`）**，以及 **backend 类型**一致性（**通常省略 `backend.type`**，由 **`service` / `handler` / `redirect`** 在无歧义时推断）：

- **`backend.type` 可选。** 省略时，若 **`service` / `handler` / `redirect`** 只有一种看起来像已配置，则自动推断类型；若多种同时存在，校验失败并要求显式写出 **`backend.type`**。
- **`backend.type` 显式写出后**，只允许与该类型匹配的块（例如 **`redirect`** 需要 **`redirect.url`**，且不得同时填充 **`service`** / **`handler`** 的配置）。

报错中会标明规则下标、配置的 host 模式以及路由 path：**`rules[N] host="..." path="..."`**。规则级 backend 使用 **`path="/"`**；子路径 backend 使用 **`paths[].path`** 中的配置字符串（若为空则退化为 `paths[index]`）。**回退（fallback）** backend 使用 **`fallback path="/"`**。

## 重新加载配置

您可以在不重启服务器的情况下重新加载配置：

1. 发送 SIGHUP 信号：`kill -HUP $(cat /tmp/gozoox.ingress.pid)`
2. 使用 reload 命令：`ingress reload`

服务器将重新加载配置文件并应用更改，而不会断开连接。
