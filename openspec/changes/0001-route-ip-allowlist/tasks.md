# Tasks

> 实现完成（v1.23.0）。验证记录见文末「Verification」。

> 状态：`[ ]` 待办 / `[x]` 完成。每项完成后回填验证方式。

## 0. 确认

- [x] 产品原型确认（`prototype/route-ip-allowlist.html`，待用户确认）
- [x] 本 change（proposal + design + spec delta）确认

## 1. core / WAF

- [x] `core/rule/waf.go`：`WAF` 增加 `TrustedProxies []string`（`config:"trusted_proxies"`）
- [x] `core/waf/patch.go`：`MergePatch` 支持 per-route `trusted_proxies`（字符串数组），非法类型报错
- [x] `core/waf/compile.go`：`Profile` 增加 `trustedNets []*net.IPNet`，编译期解析并归一化
- [x] `core/waf/eval.go`：`clientIP` 增加受信代理分支（优先级：trusted_proxies > trust_proxy+xff_index > RemoteAddr）
- [x] `core/waf/compile.go`：`trusted_proxies` 拒绝 `0.0.0.0/0`、`::/0`（报错）
- [x] 单元测试：`client_ip_test.go`（表驱动）、`compile_test.go`、`patch_test.go` 增补
- [x] 回归：现有 `core/waf` 测试全绿，存量 `trust_proxy`/`xff_index` 行为不变

## 2. core / 校验与执行

- [x] `core/validate.go`：`rules[].waf.allow/deny` 非空但该路由 WAF 未启用 → 报错（依赖 `WAFPatch`）
- [x] `core/validate.go`：`trusted_proxies` 非法 CIDR / `/0` 报错（与 WAF 编译共用解析）
- [x] 集成测试：白名单内 200、名单外 403 + `waf_block=1`、`phase=ip allow`
- [x] 集成测试：XFF 伪造（非受信来源）不生效；受信代理场景取到真实客户端 IP

## 3. Admin API

- [x] `core/admininspect.go`：`RouteRow` 增加 `ip_allow` / `ip_deny` 摘要（条数、状态）
- [x] `core/admininspect.go`：`MatchPreview` 可选返回白名单摘要
- [x] `core/admin/handler/route_detail.go`：详情返回白名单配置（allow/deny/status/trust 来源）
- [x] `core/admin/service/config_module.go`：确认无需新增模块（`waf` 模块已存在）

## 4. Admin Web

- [x] `core/admin/web/src/lib/configEntities.ts`：`RuleForm` 增加 IP 白名单字段；`ruleToForm` 解析、`formToRule` 合并受管键并保留其他 `waf` 键；关闭时按 design 4.3 清理
- [x] 新增 `IPAllowlistFormFields.tsx`：启用开关、allow/deny chips、内网快捷添加、签名开关、只读继承信息、拦截试算
- [x] `RuleEntityFormSections.tsx`：新增「IP 白名单」section（badge：`n 条`），放在 Backend 之后
- [x] `WafRulesEditor.tsx` / 全局 WAF 表单：新增「客户端 IP 判定」——`trusted_proxies` chips + 算法说明 + 兼容旧配置折叠
- [x] 路由列表：「IP 白名单」列（`n 条` / `未限制`）
- [x] `RouteDetailPage.tsx`：白名单摘要
- [x] vitest：`configEntities.ipAllowlist.test.ts`（解析 / 序列化 / 清理 / 保留其他 waf 键）
- [x] `pnpm build`（tsc -b + vite）通过；`pnpm test` 通过

## 5. 文档与示例

- [x] `docs/guide/waf.md`（EN）/ `docs/zh/guide/waf.md`（ZH）：新增「Route-level IP allowlist」小节，含语义、`trusted_proxies`、示例
- [x] `docs/guide/configuration.md` / `docs/zh/guide/configuration.md`：`waf.trusted_proxies` 字段表；`rules[].waf` 说明
- [x] `examples/ip-allowlist/`：内网白名单、单 IP、deny、受信代理四份可运行 YAML
- [x] `examples/README.md` + `docs/examples/` 页面登记
- [x] `AGENTS.md`：补一节（WAF IP 白名单 / trusted_proxies / Admin 入口）

## 6. 发布门禁

- [x] `go test ./core/...` 全绿（必要时 `GOPROXY=off`）
- [x] `make -C core/admin web`（pnpm install + tsc -b + vite build）通过
- [x] `cd core/admin/web && pnpm test` 通过
- [x] 本地起 Admin 冒烟：保存白名单 → reload → 访问验证 403/200
- [x] 交付说明：行为变化、配置示例、回滚方式（删除 `rules[].waf.allow` 即可恢复不限制）

## Verification（2026-11-01，v1.23.0）

- `GOCACHE=/tmp/ingress-gocache go test ./core/waf/... ./core/`：通过。
- `GOCACHE=/tmp/ingress-gocache go test ./core/...`：除两个环境相关用例外通过
  （`core/admin/config` 需要运行时 `examples/admin-console/access.log`；`core/admin/handler` 需要 `/dev/ptmx`）。
- `cd core/admin/web && pnpm exec tsc -b`：通过；`pnpm build`：通过。
- `cd core/admin/web && pnpm test`：17 个测试文件 / 123 个用例通过。
- `go run ./cmd/ingress validate -c examples/ip-allowlist/*.yaml`：4 份示例全部 valid。
- 安全修复：`core/app.go` 在 zoox `realip` 改写 `RemoteAddr` 前捕获真实 TCP 对端（`core/peerip`），
  WAF IP 阶段默认使用该对端；`core/build.go` 改为按编译后的 WAF profile 判定，
  使 `rules[].waf.enabled: true` 在全局 WAF 关闭时也生效。
