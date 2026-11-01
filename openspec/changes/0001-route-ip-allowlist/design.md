# design: 0001-route-ip-allowlist

> 结论先行：**不新增模块，把「域名级 IP 白名单」做成 WAF IP 阶段的一等配置入口**。配置落在 `rules[].waf.allow/deny`，客户端 IP 判定新增 `waf.trusted_proxies`，Admin 负责把「WAF 的复杂度」藏在「IP 白名单」这个产品概念后面。

## 1. 配置形态与语义

### 1.1 YAML

```yaml
# 全局（可选）：客户端 IP 判定
waf:
  trusted_proxies:
    - 10.0.0.0/8
    - 172.16.0.0/12

rules:
  - host: admin.example.com
    waf:
      enabled: true            # 该路由启用 WAF（IP 阶段才会执行）
      disable_builtin: true    # 只做 IP 准入，不跑内置攻击签名
      allow:                   # 非空 = 白名单门禁
        - 10.0.0.0/8
        - 203.0.113.7/32
      deny:                    # 可选，优先于 allow
        - 203.0.113.66/32
      block_status_code: 403   # v1 固定 403，UI 不暴露
```

### 1.2 语义表

| 配置 | 行为 |
|---|---|
| 没有 `waf` 或 `allow`/`deny` 都为空 | 不限制（**默认**，与现状完全一致） |
| `allow` 非空 | 只放行命中 `allow` 的来源；其余 403 |
| `deny` 非空 | 命中 `deny` 一律 403（即使也在 `allow` 中） |
| `allow` 空 + `deny` 非空 | 纯黑名单（高级用法，UI 折叠在「高级」里） |
| `waf.enabled: false`（且全局也没开） | 该路由 WAF 整体不执行 → 白名单不生效，**校验报错**提示补 `enabled: true` |
| `disable_builtin` 未设置 | 继承全局：全局 WAF 若启用内置签名，该域名同时跑签名（可能误伤白名单用户） |

> 语义与 WAF 现状一致（`core/waf/eval.go`：deny → allow gate → signatures），本变更**不改** IP 阶段顺序。

### 1.3 为什么复用 WAF（决策记录）

用户定版复用 WAF，不新增 `ip_access` 模块。收益与代价：

**收益**
- 零新增执行路径：IP 匹配、CIDR 解析、403 响应、日志、WAF 事件全部现成；
- 场景（scenarios）overlay 的 `rules` 合并天然覆盖 `rules[].waf`，白名单跟着场景走；
- Admin 配置落盘 / 校验 / reload 链路不用新增模块（`waf` 模块已存在）。

**代价（用 Admin UI + 校验兜住）**
- 必须显式 `waf.enabled: true` → UI 自动写入 + 校验拦截静默失效；
- 只想做 IP 白名单时会连带跑内置签名 → UI 默认写入 `disable_builtin: true`；
- 配置语义偏 WAF → UI 用「IP 白名单」产品语言包装，YAML 预览里给出最终片段。

## 2. 客户端 IP 判定（安全核心）

### 2.1 优先级

```
trusted_proxies 非空  → 受信代理算法（推荐）
否则 trust_proxy=true → 旧行为：XFF[xff_index]（兼容存量）
否则                  → RemoteAddr
```

### 2.2 受信代理算法

```text
remote = TCP 对端地址（RemoteAddr 去端口）

if trusted_proxies 为空:
    client = remote                       # 默认，最严格

if remote ∉ trusted_proxies:
    client = remote                       # 来源不可信 → 完全忽略 XFF
else:
    parts = XFF 按逗号切分、去空白、去引号、只保留合法 IP
    for ip in parts 从右到左:
        if ip ∉ trusted_proxies:
            client = ip; break            # 右起第一个非受信地址 = 真实客户端
    else:
        client = remote                   # XFF 全是受信代理（或为空）→ 退回 remote
```

要点：

- **右起**而不是左起：左起第一跳最容易被客户端伪造；右起第一个非受信地址是「离 ingress 最近的真实对端」。
- 只有 `RemoteAddr` 命中 `trusted_proxies` 才看 XFF：攻击者无法通过伪造 XFF 绕过白名单，除非他能从受信网段发起连接。
- IPv4 / IPv6 / IPv4-mapped IPv6 统一归一化后再比对（`net.IPNet.Contains` 已处理 v4-in-v6，解析时保留 `net.ParseIP`）。
- `trusted_proxies` 拒绝 `0.0.0.0/0`、`::/0`：那等于信任所有来源，白名单形同虚设（校验报错）。

### 2.3 兼容

- 存量 `waf.trust_proxy: true` + `waf.xff_index: N` 完全不变；
- 新配置两者都不用，只配 `trusted_proxies`；
- `proxy.trust_proxy` 仍然只影响**出站转发头**，不参与访问控制，避免语义漂移。

### 2.4 代码落点

| 文件 | 变化 |
|---|---|
| `core/rule/waf.go` | `WAF` 增加 `TrustedProxies []string`（tag `trusted_proxies`） |
| `core/waf/patch.go` | `MergePatch` 支持 per-route `trusted_proxies`（字符串数组） |
| `core/waf/compile.go` | `Profile` 增加 `trustedNets []*net.IPNet`，编译期解析 |
| `core/waf/eval.go` | `clientIP` 增加受信代理分支（保留旧分支） |
| `core/validate.go` | 校验 CIDR、`/0`、`allow/deny` 与 `enabled` 的一致性 |

## 3. 执行位置与顺序

复用 WAF 后，白名单在现有 WAF 调用点执行（`core/build.go`，路由匹配之后）：

```
HTTPS 强制跳转
→ 路由匹配
→ security preflight
→ rate limit
→ WAF（deny → allow gate → signatures）   ← IP 白名单在这里
→ maintenance
→ redirect / handler / upstream
```

说明：

- 白名单**在限流之后**：被拒请求仍会消耗该 IP 的限流桶。与 WAF 现状一致，可接受（每 IP 独立桶，不互相影响）。
- 白名单**在 maintenance 之前**：白名单拒绝优先于维护页。
- **健康检查 / `/_/ingress/status`** 在路由匹配前处理，天然不受白名单影响（避免 LB 探活被拦）。
- **正向代理 `connect_proxy`**：CONNECT / absolute-form 走独立入口（`core/connect_proxy.go`），v1 **不接** WAF IP 阶段（避免改代理主路径）；如域名同时是代理入口且需要白名单，后续单独评估（列在「未来扩展」）。
- **Admin 控制台 9080**：独立监听，不经过 ingress 代理，不受影响。

## 4. Admin UI 交互与 YAML 映射

### 4.1 路由编辑器新区块「IP 白名单」

位置：路由编辑抽屉的 section 列表，放在「Backend」之后、「Path 规则」之前。

| 控件 | 写入 YAML | 说明 |
|---|---|---|
| 启用开关 | `waf.enabled: true` / 删除 | 关闭时清理本功能写入的键（见 4.3） |
| 允许的 IP / CIDR（chips） | `waf.allow[]` | 必填（与 deny 至少一个）；快捷添加内网网段 |
| 高级：拒绝的 IP / CIDR | `waf.deny[]` | 折叠，默认空 |
| 同时启用 WAF 攻击签名 | `waf.disable_builtin: true`（默认）/ 不写 | 默认关闭签名，避免误伤白名单用户 |
| 客户端 IP 判定 | 只读，继承全局 | 展示「仅 RemoteAddr」或「受信代理 n 个网段」+ 跳转全局 |
| 拦截响应 | `waf.block_status_code: 403` | v1 固定 403，只读展示 |
| 拦截试算 | 不写 YAML | 输入 RemoteAddr / XFF，展示判定 IP、命中项、放行 / 拒绝 |

### 4.2 全局 WAF 模块新增「客户端 IP 判定」

- 模式单选：`仅 RemoteAddr（默认）` / `受信代理 trusted_proxies`；
- 受信代理 chips（CIDR），附「右起第一个非受信地址」的算法说明与 `/0` 警告；
- 折叠「兼容旧配置：`trust_proxy` + `xff_index`」（保留，但提示新配置优先用受信代理）；
- 展示 `waf.trusted_proxies` YAML 预览。

### 4.3 保存时的清理规则（防止残留副作用）

UI 只管理 `waf` 映射里的这几个键：`enabled`、`disable_builtin`、`allow`、`deny`、`block_status_code`；**其他键（如 `rules`、`builtin_rules`、`allow_hosts`、`log_only`）原样保留**（`formToRule(form, original)` 已做整体 spread，新增逻辑只 merge 受管键）。

关闭白名单时：

1. 删除 `allow`、`deny`；
2. 若删除后 `waf` 映射只剩本功能写入的 `enabled` / `disable_builtin` / `block_status_code`（即没有其他 WAF 配置），整个 `waf` 键删除；
3. 若还有其他 WAF 配置（如自定义签名），保留 `waf`，仅删除本功能键，避免把用户原本开启的 WAF 关掉。

### 4.4 列表 / 详情

- 路由列表新增「IP 白名单」列：`n 条`（accent badge）/ `未限制`（muted）；
- 路由详情展示：启用状态、allow/deny 条数、客户端 IP 来源、拦截状态码；
- `MatchPreview`（试匹配）可选展示该路由的白名单摘要，便于排障。

## 5. 校验与错误信息

| 场景 | 行为 | 错误信息示例 |
|---|---|---|
| `allow`/`deny` 条目非法 | fail-fast（validate / 启动 / reload / Admin 保存） | `rules[0].waf.allow["10.0.0.0/33"]: invalid CIDR` |
| `trusted_proxies` 条目非法 | 同上 | `waf.trusted_proxies["10.0.0.0/33"]: invalid CIDR` |
| `trusted_proxies` 含 `0.0.0.0/0` / `::/0` | 报错 | `waf.trusted_proxies: 0.0.0.0/0 trusts every source; IP allowlist would be bypassable` |
| per-route `allow`/`deny` 非空但合并后 `enabled=false` | 报错（`ingress validate` / Admin 保存） | `rules[0].waf.allow is set but WAF is not enabled for this rule; add rules[0].waf.enabled: true or enable global waf` |
| UI 启用但 allow/deny 都为空 | 保存按钮禁用 + 行内提示 | 白名单不能为空 |

实现注意：per-route 校验依赖 `WAFPatch`（`FinalizeLoadedConfig` 里由 `waf.ApplyRulePatchesFromYAML` 填充）。`ValidateConfig` 在 `ingress validate`、Admin `ValidateYAML`、`LoadConfigFromYAML` 路径上都能拿到 patch；直接构造 `Config` 的单测不受影响。

## 6. 拦截响应与可观测性

- 响应：WAF 默认 `block_status_code: 403` + `block_body: "Forbidden\n"`，命中 `shouldUseWAFErrorPage` 后走 `error_pages` 的 403 页面（与现有 WAF 拦截一致）。
- 访问日志：已有 `waf_block=1`，无需新增字段。
- WAF 日志：已有 `[waf block] phase=ip allow client_ip=… host=…`，能区分「IP 白名单」与「攻击签名」。
- WAF 事件：`OnWAFEvent(action, rule="ip allow", host, path, clientIP, ua)` 已入库，Admin WAF 事件列表可见；无需新建事件表。
- Admin 指标：沿用 `waf_blocks` 统计（IP 白名单拦截会计入 WAF block，语义上可接受）。

## 7. 兼容性与回归

- 不配置新字段 → 代码路径与现状完全一致；用现有 `core/waf` 测试做回归。
- 存量全局 `waf.allow` / `rules[].waf` 配置不变。
- 存量 `trust_proxy` + `xff_index` 不变。
- Admin 保存旧配置时，`rules[].waf` 若不在 UI 受管键内，原样保留。
- 场景 overlay 的 `rules` 深合并逻辑不变，`waf` 作为规则子键自动跟随。

## 8. 测试计划

**Go（core）**
- `core/waf/client_ip_test.go`（新增）：表驱动覆盖
  - 无受信代理 → RemoteAddr；
  - RemoteAddr 非受信 → 忽略 XFF；
  - RemoteAddr 受信 → 右起第一个非受信；
  - XFF 全是受信代理 → 退回 RemoteAddr；
  - 非法 XFF 段跳过；
  - 旧 `trust_proxy` + `xff_index` 行为不变。
- `core/waf/compile_test.go` 增补：`trusted_proxies` 编译、非法 CIDR、`/0` 拒绝。
- `core/waf/patch_test.go` 增补：per-route `trusted_proxies` 覆盖。
- `core/validate_test.go` 增补：`allow` 配了但未启用 → 报错。
- `core/build_test.go` / 新增集成测试：白名单内 200、名单外 403 + `waf_block=1`、XFF 伪造不生效。

**Admin web（vitest）**
- `configEntities.ipAllowlist.test.ts`：解析 / 序列化 / 清理规则 / 保留其他 waf 键；
- 试算函数（若抽成 lib）单测：与 Go 算法同表。

**手工验收**
- `ingress validate` + Admin 保存 / 发布 / reload；
- 路由列表、详情、试算、日志页展示；
- 场景切换后白名单跟随。

## 9. 风险与边界

| 风险 | 处理 |
|---|---|
| XFF 伪造绕过 | 默认只认 RemoteAddr；受信代理才看 XFF；拒绝 `/0` |
| 受信代理网段配错（太宽） | UI 警告 + 文档说明；`/0` 直接报错 |
| 配了 allow 但忘了 enabled → 静默不生效 | 校验报错 + UI 自动写 `enabled: true` |
| 只想白名单却误开内置签名导致误伤 | UI 默认 `disable_builtin: true` |
| 多级代理 / CDN 回源 | 受信代理需把 CDN 回源网段也加入；按右起算法取真实客户端 |
| 被拒请求消耗限流桶 | 与 WAF 现状一致，接受；文档说明 |
| IPv6 / v4-mapped | 解析归一化后比对；测试覆盖 |
| 关闭白名单残留 `enabled: true` 改变 WAF 行为 | 4.3 清理规则 + 单测 |

## 10. 未来扩展（不在本次范围）

- `paths[]` 路径级白名单；
- 404 隐藏（需要 WAF 返回拦截阶段，改造 `CheckRequest` 返回值）；
- 正向代理 `connect_proxy` 入口接入 IP 阶段；
- GeoIP / 国家 / ASN 维度；
- 临时授权（带过期时间）与审批流；
- 独立的「访问控制」事件表与指标卡（当前复用 WAF 事件）。
