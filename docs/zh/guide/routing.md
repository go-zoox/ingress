# 路由

Ingress 提供灵活的路由功能来匹配请求并将它们路由到相应的后端服务。您可以按主机名和路径匹配请求，支持精确、正则表达式和通配符匹配。

## 主机匹配

主机匹配是路由请求的主要方式。Ingress 支持三种主机匹配方式：**精确（exact）**、**正则（regex）**、**通配符（wildcard）**。

### 自动 `host_type`（默认）

若**省略** `host_type` 或设为 `host_type: auto`，Ingress 会在**编译路由索引时**（进程启动或配置 Reload）根据 `host` 字符串自动选择匹配类型：

1. 若 `host` 中含正则元字符 `( ) [ ] ^ $ | + ? \` → 按 **regex** 处理  
2. 否则若含 `*` → 按 **wildcard** 处理  
3. 否则 → **exact**

会先判断正则再判断 `*`，因此像 `^.*\.example\.com$` 这类完整正则不会被误判为通配符。

解析后的类型会写回规则上的 `host_type`，供后续逻辑使用（如 `service.name` 捕获、错误页分支等）。若必须按**字面量**匹配 `host`（即使看起来像模式），请显式写 **`host_type: exact`**。

省略 `host_type` 的示例：

```yaml
rules:
  # 编译为 regex（括号、\w 等）
  - host: ^([a-z0-9-]+)\.inlets\.example\.com$
    backend:
      service:
        name: inlets
        port: 8080
  # 编译为 wildcard
  - host: '*.api.example.com'
    backend:
      service:
        name: api-gateway
        port: 8080
  # 编译为 exact
  - host: idp.example.com
    backend:
      service:
        name: idp
        port: 443
```

### 精确匹配

精确匹配按字面量完全匹配主机名。在自动 `host_type` 下，普通主机名（无正则元字符且无 `*`）会解析为 **exact**：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
```

这将精确匹配 `Host: example.com` 的请求。

### 正则表达式匹配

正则表达式匹配允许您使用正则表达式来匹配主机名。可显式写 `host_type: regex`，也可在模式中含正则元字符时省略 `host_type`，由编译阶段自动推断为 regex。

```yaml
rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
```

在此示例中，`$1` 引用 host 正则中的第一个捕获组。对 `t-myapp.example.work` 的请求将被路由到 `task.myapp.svc`。

**说明：** 在 Go 的 `regexp` 中，`\w` 等价于 `[0-9A-Za-z_]`，**不包含**连字符 `-`。若子域带横线（如 `my-app.example.work`），需使用允许 `-` 的写法，例如 `^t-([a-zA-Z0-9-]+).example.work`，不能仅依赖 `(\w+)`。

### Service 名称捕获模板

当使用 `host_type: regex` 时，也可以在 `service.name` 中使用带作用域的捕获模板（高级用法）：

- `${host.<索引>}`：来自 host 正则的捕获组
- `${path.<索引>}`：来自命中 path 正则的捕获组

```yaml
rules:
  - host: ^t-(\w+)-(dev|prod).example.work$
    host_type: regex
    backend:
      service:
        name: task.${host.1}.${host.2}.svc
        port: 8080
    paths:
      - path: ^/api/v1/([^/]+)/([^/]+)$
        backend:
          service:
            name: ${path.2}.${path.1}.${host.2}.${host.1}.svc
            port: 8080
```

兼容性说明：

- `service.name` 中旧写法 `$1`、`$2`...（基于 host 正则）是默认优先/基础用法，并持续完全兼容。
- `request.path.rewrites` 仍使用重写语法里的 `$1`、`$2`...（例如 `^/api/(.*):/v2/$1`）。

### 通配符匹配

通配符匹配使用 `*` 作为通配符。可显式写 `host_type: wildcard`，也可在 `host` 含 `*` 且无正则元字符时省略 `host_type`（见上文「自动 `host_type`」一节）。

```yaml
rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

这将匹配 `example.work` 的任何子域，例如 `app.example.work`、`api.example.work` 等。

## 基于路径的路由

您可以在主机规则内定义基于路径的路由规则。路径使用正则表达式模式匹配：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: default-service
        port: 8080
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8080
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8080
```

路径匹配使用正则表达式模式。路径 `/api` 将匹配 `/api`、`/api/`、`/api/users` 等。

### 路径匹配优先级

路径按定义的顺序匹配。将使用第一个匹配的路径。如果没有路径匹配，将使用主机级后端。

## 请求重写

在路由到后端服务时，您可以重写请求路径、头和查询参数。

### 路径重写

使用正则表达式模式重写请求路径：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          path:
            rewrites:
              - ^/api/v1/(.*):/api/v2/$1
              - ^/old:/new
```

重写格式为 `pattern:replacement`，其中 `pattern` 是正则表达式，`replacement` 是新路径。可以使用 `$1`、`$2` 等引用捕获组。

### Host 头重写

上游 **`Host`** 由可选的 **`service.request.host.rewrite`** 与省略 **`rewrite`** 时的 **`backend.service.mode`**（兼容 **`backend.mode`**）共同决定。完整说明与 **fallback** 行为见 **[请求和响应重写](./rewriting.md)**。

显式 `rewrite` 示例：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          host:
            rewrite: true
```

反代公网 HTTPS 源站时推荐 `mode`：

```yaml
rules:
  - host: mirror.example.com
    backend:
      service:
        mode: external
        protocol: https
        name: upstream.example.org
```

### 头修改

添加或修改请求头：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          headers:
            X-Forwarded-Proto: https
            X-Custom-Header: value
```

### 查询参数修改

添加或修改查询参数：

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: backend-service
        port: 8080
        request:
          query:
            api_key: secret-key
            version: v2
```

## 重定向

使用 **`backend.redirect`** 直接返回重定向而不走反向代理。**`backend.type` 可选**：当该 `backend` **仅配置了 `redirect`** 时，Ingress 会自动推断为 **`redirect`**，通常 **省略 `backend.type`**。允许的显式值为 **`service`**、**`handler`**、**`redirect`**；同一 `backend` 只能填充与类型对应的配置块。若 **`service`** / **`handler`** / **`redirect`** 在省略 `type` 时 **看起来同时存在**，`ingress validate` 会失败，直到你 **显式写出 `backend.type`**。

下面两条规则推断结果一致——对照 **`type: redirect`** 与省略 **`type`**：

```yaml
rules:
  - host: old-explicit.example.com
    backend:
      type: redirect
      redirect:
        url: https://new.example.com
        permanent: true
  - host: old-inferred.example.com
    backend:
      redirect:
        url: https://new.example.com
        permanent: true
```

可运行的双 host 对照：`examples/ssl-tls/route-redirect.yaml`。

字段说明：

- **`url`**：跳转地址。若非 `http://` / `https://` 开头，则视为主机（可含端口），Ingress 会用当前请求的协议，并保留原始 path 与 query 拼出完整 URL。
- **`permanent`**：为 `false` 时使用 **302**，为 `true` 时使用 **301**；若开启下面的 `with_origin_method_and_body`，则状态码见该项说明。
- **`with_origin_method_and_body`**（默认 `false`）：为 `true` 时使用 **307** / **308**，客户端会保留原 HTTP 方法与请求体（临时/永久仍由 `permanent` 决定）；为 `false` 时仍为 **302** / **301**。

可在 **`url` 中写捕获占位**，规则与 `service.name` 一致：`${host.N}`、`${path.N}`；对正则或通配 host 还可使用基于 host 模式的 **`$1` 风格** 替换。若重定向来自某条已匹配的 `paths[].path`，可使用 path 捕获。

```yaml
rules:
  - host: '^bigscreen-([^.]+)\.ys\.example\.com$'
    host_type: regex
    backend:
      type: redirect
      redirect:
        url: https://bigscreen-$1.other.example.com
```

同一 host 上「默认重定向 + 按 path 反代 / 按 path 再重定向」可参考 **`examples/redirect/capture-and-mixed.yaml`**：其中 **部分 backend 显式写 `backend.type`，部分省略**，便于在同一文件里对照。

全局 HTTP→HTTPS 强跳使用 `https.redirect_from_http`（同样支持 `with_origin_method_and_body`），详见 [SSL/TLS 指南](/zh/guide/ssl-tls)。

## Handler 后端

路径级 `backend` 也可以使用 **`backend.handler`** 直接响应。**`backend.type` 可选**：当 **仅配置了 `handler`** 时会推断为 **`handler`**。下方示例仅在第一条 path 上保留 **`type: handler`**，其余 path 省略 **`backend.type`**，便于对照。

**可运行示例**（含 `file_server`、`templates`、`script`）见 **`examples/handler/`** — [Handler 示例](/zh/examples/handler)。

通过 **`handler.type`** 选择：

- `static_response`（默认）
- `file_server`
- `templates`
- `script`

```yaml
rules:
  - host: handler.example.com
    backend:
      service:
        name: api-service
        port: 8080
    paths:
      - path: /custom/handler/json
        backend:
          type: handler
          handler:
            type: static_response
            status_code: 200
            headers:
              Content-Type: application/json
            body: |
              {"message":"Hello, World!"}
      - path: /custom/handler/files
        backend:
          handler:
            type: file_server
            root_dir: /app/public
            index_file: index.html
      - path: /custom/handler/templates
        backend:
          handler:
            type: templates
            root_dir: /app/templates
      - path: /custom/handler/script/js
        backend:
          handler:
            type: script
            engine: javascript
            script: |
              ctx.response.status_code = 200
              ctx.type = "application/json"
              ctx.body = JSON.stringify({ method: ctx.method, path: ctx.path })
              ctx.setHeader("X-Handler-Engine", "javascript")
      - path: /custom/handler/script/go
        backend:
          handler:
            type: script
            engine: go
            script: |
              ctx.SetHeader("X-Handler-Engine", "go")
              ctx.String(200, "%s %s", ctx.Method, ctx.Path)
```

- `backend.type`：可选——在无歧义时由已配置块推断 **`service`** / **`handler`** / **`redirect`**；仅当 **`ingress validate`** 提示歧义时再显式写出
- **`backend.service.mode`**：`internal`（默认）或 `external`——未设置 **`service.request.host.rewrite`** 时发往上游的 **`Host`** 默认行为（**`external`** 将 **`Host`** 对齐到 **`service.name`**；见[重写](./rewriting.md)）。仍可使用与之一致的 **`backend.mode`**。
- `handler.type`: `static_response`（默认）、`file_server`、`templates`、`script`
- 当 `handler.type=static_response`：支持 `status_code`、`headers`、`body`
- 当 `handler.type=file_server`：支持 `root_dir`、`index_file`
- 当 `handler.type=templates`：支持 `root_dir`
- 当 `handler.type=script`：支持 `engine`、`script`
  - `engine=javascript`：使用 `goja`；提供 `ctx`：
    - `ctx.request` / `ctx.response`
    - 别名：`ctx.method`、`ctx.path`、`ctx.headers`
    - 响应别名：`ctx.status`（`ctx.response.status_code`）、`ctx.type`（`ctx.response.content_type`）、`ctx.body`（`ctx.response.body`）
    - 方法：`ctx.setHeader(key, value)` 和 `ctx.response.setHeader(key, value)`
  - `engine=go`：使用 `yaegi` 执行脚本，`ctx` 为原生 `*zoox.Context`（例如：`ctx.SetHeader(...)`、`ctx.String(...)`、`ctx.Fetch()`）

## 回退服务

如果没有规则匹配请求，则使用回退服务：

```yaml
fallback:
  service:
    name: fallback-service
    port: 8080
```

回退服务对于处理未匹配的请求或提供默认后端很有用。若未设置 **`fallback.service.request.host.rewrite`**，发往 fallback 的 **`Host`** 会对齐到该回退服务（见[重写](./rewriting.md)）。

## 路由示例

### 同一主机上的多个服务

```yaml
rules:
  - host: example.com
    backend:
      service:
        name: web-service
        port: 8080
    paths:
      - path: /api
        backend:
          service:
            name: api-service
            port: 8081
      - path: /admin
        backend:
          service:
            name: admin-service
            port: 8082
```

### 带路径重写的正则主机

```yaml
rules:
  - host: ^t-(\w+).example.work
    host_type: regex
    backend:
      service:
        name: task.$1.svc
        port: 8080
    paths:
      - path: /api/v1/([^/]+)
        backend:
          service:
            name: $1.example.work
            port: 8080
            request:
              path:
                rewrites:
                  - ^/api/v1/([^/]+):/api/v1/task/$1
```

### 通配符主机匹配

```yaml
rules:
  - host: '*.example.work'
    host_type: wildcard
    backend:
      service:
        name: wildcard-service
        port: 8080
```

这匹配 `example.work` 的任何子域并路由到同一个后端服务。

## 匹配如何构建（预编译）

Ingress **不会**在每次请求时再为 host、path 解析正则。

- 进程**启动**或配置 **Reload** 时，`prepare()` 会构建内部**路由索引**（`core/compile.go`）：对每条规则先解析最终 `host_type`（含省略或 `auto` 时的**自动推断**），再对作为 `regex` / `wildcard` 的 `host` 以及每条 `paths[].path`，使用 Go `regexp` **只编译一次**。
- **配置里规则的顺序会保留。** 匹配按规则顺序遍历；**先**命中的 host 规则生效；在同一 host 下 **先**命中的 path 生效（与优化前语义一致）。
- 若存在**非法**模式（例如 `host` 或 `path` 的正则无法编译），**启动或 `Reload` 会直接报错失败**，需先修正配置。这与早期「可能直到第一次匹配请求才暴露错误」的行为不同。

请求路径上实际仍使用预编译索引。若启用缓存，**按主机**缓存的路由结果可能以 `match.host:v2:<hostname>` 形式的键保存（见 [缓存](./caching.md)），直至 `cache.ttl` 过期。

## 最佳实践

1. **顺序很重要**：将更具体的规则放在通用规则之前
2. **尽可能使用精确匹配**：普通主机名会推断为 exact，通常比正则或通配符更快
3. **需要时可省略 `host_type` 或写 `auto`**：看起来像正则或通配符的 `host` 会在编译阶段自动识别；若需覆盖（例如含 `*` 或括号但必须按字面量匹配），请显式写 `host_type`
4. **测试正则表达式模式**：确保模式符合预期；非法模式会在启动或重载阶段失败
5. **使用路径路由**：按路径组织路由以提高可维护性
6. **设置回退**：始终为未匹配的请求配置回退服务
