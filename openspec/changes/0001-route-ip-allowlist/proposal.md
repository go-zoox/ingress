# change: 0001-route-ip-allowlist（域名级 IP 白名单，复用 WAF IP 阶段）

## 为什么

当前域名路由没有任何来源 IP 准入能力：

- 内网 / 管理类域名（admin、内部 API、灰度、运维入口）一旦解析到公网，只能靠认证兜底，暴露面不可控；
- 用户诉求很明确：**默认不限制**，个别域名只允许内网或指定 IP 访问，避免安全问题。

现有代码里其实已经有可复用的能力：WAF 的 IP 阶段（`core/waf/eval.go`）先判 `deny`、再判 `allow`（`allow` 非空 = 只放行名单内 IP），并且支持按域名用 `rules[].waf` 打补丁（`core/waf/compile.go`、`core/waf/patch.go`）。但目前：

1. 只有**全局** `waf.allow/deny` 有 Admin UI；单条路由的 `rules[].waf` 只能手写 YAML / 场景 overlay；
2. 直接写 `rules[].waf.allow` 需要同时理解 `enabled`、`disable_builtin`、`trust_proxy`、`block_status_code`，很容易配出一个「看起来配了、其实没生效」或「误伤白名单用户」的配置；
3. 客户端 IP 判定目前只有 `trust_proxy` 布尔 + `xff_index`，开了就整条 XFF 可信——ingress 直接暴露公网时，伪造 `X-Forwarded-For` 即可绕过白名单。

**用户定版：复用 WAF 的 `allow`，配置入口放在单条路由上；不新增顶层 `ip_access` 模块。**

## 做什么

### 1. 路由级 IP 白名单（复用 WAF）

- 在 Admin 路由编辑器新增「IP 白名单」区块，写 `rules[].waf.allow/deny`；
- 语义沿用 WAF IP 阶段：
  - `allow` 非空 ⇒ 只有名单内 IP / CIDR 放行；
  - `deny` 优先于 `allow`，命中即拒；
  - 两者都空 ⇒ 不限制（默认行为，不改变现有配置）；
- 拦截响应固定 **403**，走现有 `error_pages` 的 403 页面（不新增响应体）。

### 2. 客户端 IP 判定：受信代理网段

- 新增全局 `waf.trusted_proxies`（CIDR 列表）；
- 默认（空）只认 TCP `RemoteAddr`；
- 配置后：仅当 `RemoteAddr ∈ trusted_proxies` 才采信 `X-Forwarded-For`，并取**右起第一个非受信地址**作为客户端 IP；
- 保留旧 `trust_proxy` + `xff_index` 兼容，存量配置行为不变。

### 3. Admin 控制台

- 路由列表新增「IP 白名单」列（`n 条` / `未限制`）；
- 路由详情展示白名单摘要与拦截状态码；
- 全局 WAF 模块新增「受信代理 trusted_proxies」编辑；
- 路由编辑器内提供「拦截试算」：输入 RemoteAddr / XFF，直接看放行 / 拒绝与判定依据。

### 4. 校验与可观测性

- 非法 IP / CIDR 在 `ingress validate`、启动、reload、Admin 保存时 fail-fast；
- `rules[].waf.allow/deny` 配了但该路由 WAF 未启用时校验报错（避免安全配置静默失效）；
- `trusted_proxies` 含 `0.0.0.0/0` / `::/0` 报错（等于信任所有来源，白名单失效）；
- 沿用现有可观测性：WAF 日志 `[waf block] phase=ip allow`、访问日志 `waf_block=1`、WAF 事件 `rule=ip allow`。

## 配置示例

```yaml
waf:
  trusted_proxies:            # 可选；默认空 = 只认 RemoteAddr
    - 10.0.0.0/8              # 例如集群入口 LB 的网段
    - 172.16.0.0/12

rules:
  - host: admin.example.com
    backend:
      service:
        name: admin-console
        port: 8080
    waf:
      enabled: true           # 单条路由启用 WAF（IP 阶段随之生效）
      disable_builtin: true   # 只做 IP 准入，不跑内置攻击签名
      allow:
        - 10.0.0.0/8
        - 192.168.0.0/16
        - 203.0.113.7/32
      # deny:                 # 可选，优先于 allow
      #   - 203.0.113.66/32
```

## 不做什么

- **不新增顶层 `ip_access` 模块**（用户定版：复用 WAF）；
- **不做 path 级白名单**：v1 只做域名级（`rules[]`），`paths[]` 后续再评估；
- **不做 404 隐藏**：v1 固定 403（404 需要 WAF 返回拦截阶段，属后续改造）；
- 不做 GeoIP / 国家白名单、临时授权 / 审批流、独立 IP 拦截事件表；
- 不改变现有全局 `waf.allow/deny` 与攻击签名语义；
- 不改变 `proxy.trust_proxy`（它只管出站转发头，与访问控制解耦）。

## 影响面

| 层 | 文件（现状） | 变化 |
|---|---|---|
| core rule | `core/rule/waf.go` | 新增 `TrustedProxies []string` |
| core waf | `core/waf/compile.go`、`eval.go`、`patch.go` | 编译受信网段；客户端 IP 新算法；patch 支持 `trusted_proxies` |
| core validate | `core/validate.go` | 新增 IP 白名单相关校验 |
| core admin api | `core/admininspect.go`、`core/admin/handler/route_detail.go` | 路由行 / 试匹配 / 详情增加白名单摘要 |
| admin web | `RuleEntityFormSections.tsx`、新增 `IPAllowlistFormFields.tsx`、`configEntities.ts`、`WafRulesEditor.tsx`、`RouteDetailPage.tsx` | 路由编辑新区块、全局受信代理、列表列、详情 |
| docs / examples | `docs/guide/waf.md`、`docs/guide/configuration.md`、`docs/zh/...`、`examples/` | 使用说明与可运行示例 |
| tests | `core/waf/*_test.go`、`core/validate_test.go`、`core/admin/web/src/lib/*.test.ts` | 表驱动单测 + 回归 |

## 验收

1. **默认不配置**：所有既有路由行为不变（回归测试）。
2. `admin.example.com` 配 `allow: [10.0.0.0/8]`：内网 IP 200，公网 IP 403（`error_pages.403`），访问日志 `waf_block=1`，WAF 日志 `phase=ip allow`。
3. 配 `trusted_proxies` 后：来自非受信来源的伪造 XFF **不生效**（按 RemoteAddr 判定）；来自受信代理的 XFF 按「右起第一个非受信地址」取到真实客户端 IP。
4. 非法 CIDR / 受信代理写 `/0` / `allow` 配了但路由 WAF 未启用：`ingress validate` 与 Admin 保存均报错。
5. Admin：路由编辑保存后 YAML 正确、刷新后回显；关闭白名单后不残留 `enabled` / `disable_builtin` 等副作用字段；YAML 里已有的 `waf.rules` 自定义签名不被覆盖。
6. 场景（scenarios）overlay 合并 `rules` 时，`rules[].waf` 随规则一起合并，白名单在场景切换后生效。
7. 文档与示例可运行：`ingress validate -c examples/ip-allowlist/*.yaml` 通过。
