import {
  CollapsibleFormSection,
  FormCheckbox,
  FormField,
  FormGrid,
  FormSection,
  FormSelectField,
} from './Form'
import { AdminDatabaseFormFields } from './config/AdminDatabaseFormFields'
import { AdminGeoIPFormFields } from './config/AdminGeoIPFormFields'
import { FallbackEditor } from './config/FallbackEditor'
import { RulesEditor } from './config/RulesEditor'
import { ServicesEditor } from './config/ServicesEditor'
import { RateLimitFormFields } from './config/RateLimitFormFields'
import { SecurityFormFields } from './config/SecurityFormFields'
import { SslCertsEditor } from './config/SslCertsEditor'
import { WafRulesEditor } from './config/WafRulesEditor'
import { ErrorPagesFormFields } from './config/ErrorPagesFormFields'
import { GlobalMaintenanceFormFields } from './config/GlobalMaintenanceFormFields'
import {
  globalMaintenanceFromDoc,
  patchGlobalMaintenance,
  type GlobalMaintenanceForm,
} from '../lib/maintenance'
import {
  WAF_GLOBAL_MODE_OPTIONS,
  type WAFGlobalMode,
  wafGlobalModeFromLogOnly,
} from '../lib/wafAction'
import {
  bool,
  num,
  obj,
  arr,
  parseModuleDoc,
  setBool,
  setNum,
  setStr,
  str,
  stringifyModuleDoc,
} from '../lib/ingressModuleForms'
import {
  patchGlobalRateLimit,
  rateLimitFromDoc,
  type RateLimitFormSlice,
  patchGlobalSecurity,
  securityFromDoc,
  type SecurityFormSlice,
} from '../lib/configEntities'

function GeneralModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const patch = (fn: (next: Record<string, unknown>) => void) => {
    const next = { ...doc }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid>
      <FormField
        label="HTTP 端口"
        keyName="port"
        hint="明文 HTTP 监听端口"
        type="number"
        value={num(doc.port, 8080)}
        onChange={(e) => patch((n) => setNum(n, 'port', Number(e.target.value)))}
      />
      <FormSection title="高级选项">
        <FormCheckbox
          label="启用 H2C（明文 HTTP/2，仅内网/测试）"
          checked={bool(doc.enable_h2c)}
          onChange={(v) => patch((n) => setBool(n, 'enable_h2c', v))}
        />
        <FormCheckbox
          label="错误页暴露请求细节（内置模板）"
          checked={bool(doc.error_page_expose_details)}
          onChange={(v) => patch((n) => setBool(n, 'error_page_expose_details', v))}
        />
      </FormSection>
      <ErrorPagesFormFields doc={doc} onChange={onChange} />
    </FormGrid>
  )
}

function CacheModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const cache = { ...obj(doc.cache) }
  const patchCache = (fn: (next: Record<string, unknown>) => void) => {
    const nextCache = { ...cache }
    fn(nextCache)
    onChange({ cache: nextCache })
  }

  return (
    <FormGrid>
      <FormField
        label="TTL（秒）"
        keyName="ttl"
        hint="全局 matcher / 缓存键默认过期时间"
        type="number"
        value={num(cache.ttl, 300)}
        onChange={(e) => patchCache((n) => setNum(n, 'ttl', Number(e.target.value)))}
      />
      <FormField
        label="Redis 主机"
        keyName="host"
        value={str(cache.host)}
        onChange={(e) => patchCache((n) => setStr(n, 'host', e.target.value))}
      />
      <FormField
        label="Redis 端口"
        keyName="port"
        type="number"
        value={num(cache.port, 6379)}
        onChange={(e) => patchCache((n) => setNum(n, 'port', Number(e.target.value)))}
      />
      <FormField
        label="键前缀"
        keyName="prefix"
        value={str(cache.prefix)}
        onChange={(e) => patchCache((n) => setStr(n, 'prefix', e.target.value))}
      />
      <FormSection title="认证（可选）">
        <FormField
          label="用户名"
          keyName="username"
          value={str(cache.username)}
          onChange={(e) => patchCache((n) => setStr(n, 'username', e.target.value))}
        />
        <FormField
          label="密码"
          keyName="password"
          type="password"
          value={str(cache.password)}
          onChange={(e) => patchCache((n) => setStr(n, 'password', e.target.value))}
        />
        <FormField
          label="DB 编号"
          keyName="db"
          type="number"
          value={num(cache.db, 0)}
          onChange={(e) => patchCache((n) => setNum(n, 'db', Number(e.target.value)))}
        />
      </FormSection>
    </FormGrid>
  )
}

function RateLimitModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const form = rateLimitFromDoc(doc)
  const onFormChange = (next: RateLimitFormSlice) => {
    onChange(patchGlobalRateLimit(doc, next))
  }

  return (
    <RateLimitFormFields
      form={form}
      onChange={onFormChange}
      title="全局限流 rate_limit"
    />
  )
}

function SecurityModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const form = securityFromDoc(doc)
  const onFormChange = (next: SecurityFormSlice) => {
    onChange(patchGlobalSecurity(doc, next))
  }

  return (
    <SecurityFormFields
      form={form}
      onChange={onFormChange}
      title="全局 security"
    />
  )
}

function AdminModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const admin = { ...obj(doc.admin) }
  const database = { ...obj(admin.database) }
  const geoip = { ...obj(admin.geoip) }
  const web = { ...obj(admin.web) }
  const auth = { ...obj(admin.auth) }
  const authBasic = { ...obj(auth.basic) }
  const authOAuth = { ...obj(auth.oauth) }
  const authType = str(auth.type, 'basic') || 'basic'

  const patchAdmin = (fn: (next: Record<string, unknown>) => void) => {
    const nextAdmin = { ...admin }
    fn(nextAdmin)
    onChange({ admin: nextAdmin })
  }

  return (
    <FormGrid>
      <FormCheckbox
        label="启用 Admin 控制台 admin.enabled"
        checked={bool(admin.enabled)}
        onChange={(v) => patchAdmin((n) => setBool(n, 'enabled', v))}
      />
      <FormField
        label="Admin 端口"
        keyName="port"
        type="number"
        value={num(admin.port, 9080)}
        onChange={(e) => patchAdmin((n) => setNum(n, 'port', Number(e.target.value)))}
      />
      <FormSection title="日志路径（可选）">
        <p className="form-hint">
          留空时使用 <code>logging</code> 段配置；若 logging 也未指定且 admin 已启用，则默认写入
          ingress.yaml 同目录的 <code>access.log</code> / <code>error.log</code>。
        </p>
        <FormField
          label="Access 日志 admin.access_log_path"
          value={str(admin.access_log_path)}
          onChange={(e) => patchAdmin((n) => setStr(n, 'access_log_path', e.target.value))}
        />
        <FormField
          label="Error 日志 admin.error_log_path"
          value={str(admin.error_log_path)}
          onChange={(e) => patchAdmin((n) => setStr(n, 'error_log_path', e.target.value))}
        />
      </FormSection>
      <FormSection title="数据库">
        <AdminDatabaseFormFields
          driver={str(database.driver, 'sqlite')}
          dsn={str(database.dsn, 'file:./admin.db?cache=shared&_fk=1')}
          onChange={({ driver: nextDriver, dsn: nextDsn }) =>
            patchAdmin((n) => {
              const nextDB = { ...obj(n.database) }
              setStr(nextDB, 'driver', nextDriver)
              setStr(nextDB, 'dsn', nextDsn)
              n.database = nextDB
            })
          }
        />
      </FormSection>
      <FormSection title="GeoIP（WAF 攻击地图）">
        <AdminGeoIPFormFields
          geoip={geoip}
          onChange={(nextGeoIP) =>
            patchAdmin((n) => {
              if (Object.keys(nextGeoIP).length === 0) {
                delete n.geoip
              } else {
                n.geoip = nextGeoIP
              }
            })
          }
        />
      </FormSection>
      <FormSection title="认证">
        <FormSelectField
          label="认证方式 admin.auth.type"
          hint="默认 basic；none 关闭登录，oauth 使用第三方登录"
          value={authType}
          onChange={(e) =>
            patchAdmin((n) => {
              const nextAuth = { ...obj(n.auth) }
              setStr(nextAuth, 'type', e.target.value)
              n.auth = nextAuth
            })
          }
        >
          <option value="basic">basic（本地账号）</option>
          <option value="none">none（无需登录）</option>
          <option value="oauth">oauth（第三方登录）</option>
        </FormSelectField>
        {authType === 'basic' ? (
          <>
            <FormField
              label="初始管理员用户名 admin.auth.basic.username"
              hint="对应 RBAC 超级管理员（admin 角色）；启动时会自动创建或补全该用户"
              value={str(authBasic.username, 'admin')}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextBasic = { ...obj(nextAuth.basic) }
                  setStr(nextBasic, 'username', e.target.value)
                  nextAuth.basic = nextBasic
                  n.auth = nextAuth
                })
              }
            />
            <FormField
              label="初始管理员密码 admin.auth.basic.password"
              type="password"
              value={str(authBasic.password)}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextBasic = { ...obj(nextAuth.basic) }
                  setStr(nextBasic, 'password', e.target.value)
                  nextAuth.basic = nextBasic
                  n.auth = nextAuth
                })
              }
            />
          </>
        ) : null}
        {authType === 'oauth' ? (
          <>
            <FormField
              label="Provider admin.auth.oauth.provider"
              hint="github / google / gitlab / microsoft 等"
              value={str(authOAuth.provider)}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextOAuth = { ...obj(nextAuth.oauth) }
                  setStr(nextOAuth, 'provider', e.target.value)
                  nextAuth.oauth = nextOAuth
                  n.auth = nextAuth
                })
              }
            />
            <FormField
              label="Client ID admin.auth.oauth.client_id"
              value={str(authOAuth.client_id)}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextOAuth = { ...obj(nextAuth.oauth) }
                  setStr(nextOAuth, 'client_id', e.target.value)
                  nextAuth.oauth = nextOAuth
                  n.auth = nextAuth
                })
              }
            />
            <FormField
              label="Client Secret admin.auth.oauth.client_secret"
              type="password"
              value={str(authOAuth.client_secret)}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextOAuth = { ...obj(nextAuth.oauth) }
                  setStr(nextOAuth, 'client_secret', e.target.value)
                  nextAuth.oauth = nextOAuth
                  n.auth = nextAuth
                })
              }
            />
            <FormField
              label="Redirect URL admin.auth.oauth.redirect_url"
              hint="留空则自动使用 /api/v1/auth/oauth/callback"
              value={str(authOAuth.redirect_url)}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextOAuth = { ...obj(nextAuth.oauth) }
                  setStr(nextOAuth, 'redirect_url', e.target.value)
                  nextAuth.oauth = nextOAuth
                  n.auth = nextAuth
                })
              }
            />
            <FormField
              label="Scopes admin.auth.oauth.scopes"
              hint="逗号分隔，留空使用 provider 默认值"
              value={arr<string>(authOAuth.scopes).join(', ')}
              onChange={(e) =>
                patchAdmin((n) => {
                  const nextAuth = { ...obj(n.auth) }
                  const nextOAuth = { ...obj(nextAuth.oauth) }
                  const scopes = e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean)
                  if (scopes.length > 0) {
                    nextOAuth.scopes = scopes
                  } else {
                    delete nextOAuth.scopes
                  }
                  nextAuth.oauth = nextOAuth
                  n.auth = nextAuth
                })
              }
            />
          </>
        ) : null}
      </FormSection>
      <FormSection title="Web UI">
        <FormCheckbox
          label="Dev 代理 admin.web.dev_proxy（仅 API，UI 用 pnpm dev）"
          checked={bool(web.dev_proxy)}
          onChange={(v) =>
            patchAdmin((n) => {
              const nextWeb = { ...obj(n.web) }
              setBool(nextWeb, 'dev_proxy', v)
              n.web = nextWeb
            })
          }
        />
      </FormSection>
    </FormGrid>
  )
}

function LoggingModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const DEFAULT_ACCESS_LOG = '/var/log/ingress/access.log'
  const DEFAULT_ERROR_LOG = '/var/log/ingress/error.log'

  const logging = { ...obj(doc.logging) }
  const transports = arr<Record<string, unknown>>(logging.transports)
  const fileIdx = transports.findIndex((t) => str(t.type) === 'file')
  const file = fileIdx >= 0 ? { ...obj(transports[fileIdx]) } : { type: 'file', path: '', levels: {} }
  const levels = { ...obj(file.levels) }
  const hasCustomPaths = fileIdx >= 0 && (Boolean(str(file.path)) || Boolean(str(levels.error)))
  const enabled = logging.enable === undefined
    ? hasCustomPaths
    : bool(logging.enable)

  const patchLogging = (fn: (next: Record<string, unknown>) => void) => {
    const nextLogging = { ...logging }
    fn(nextLogging)
    onChange({ logging: nextLogging })
  }

  const setEnabled = (v: boolean) => {
    patchLogging((n) => {
      if (v) {
        n.enable = true
        if (!hasCustomPaths) delete n.transports
      } else {
        n.enable = false
        delete n.transports
      }
    })
  }

  const patchFile = (accessPath: string, errorPath: string) => {
    const access = accessPath.trim()
    const errPath = errorPath.trim()
    patchLogging((n) => {
      n.enable = true
      if (!access && !errPath) {
        delete n.transports
        return
      }
      const nextFile: Record<string, unknown> = {
        type: 'file',
        path: access,
        levels: errPath ? { error: errPath } : {},
      }
      const nextTransports = [...transports]
      if (fileIdx >= 0) nextTransports[fileIdx] = nextFile
      else nextTransports.push(nextFile)
      n.transports = nextTransports
    })
  }

  const accessDisplay = str(file.path) || (enabled && !hasCustomPaths ? DEFAULT_ACCESS_LOG : '')
  const errorDisplay = str(levels.error) || (enabled && !hasCustomPaths ? DEFAULT_ERROR_LOG : '')

  return (
    <FormGrid>
      <FormCheckbox
        label="启用文件日志 logging.enable"
        checked={enabled}
        onChange={setEnabled}
      />
      <FormSelectField
        label="日志级别"
        keyName="level"
        value={str(logging.level, 'warn')}
        onChange={(e) => patchLogging((n) => setStr(n, 'level', e.target.value))}
      >
        <option value="debug">debug</option>
        <option value="info">info</option>
        <option value="warn">warn</option>
        <option value="error">error</option>
      </FormSelectField>
      {enabled && (
        <FormSection title="文件输出">
          <p className="form-hint">
            启用 <code>admin.enabled</code> 且此处未配置时，运行时默认开启文件日志并在 ingress.yaml
            同目录写入 <code>access.log</code> / <code>error.log</code>。未启用 admin 时，留空默认{' '}
            <code>{DEFAULT_ACCESS_LOG}</code>、<code>{DEFAULT_ERROR_LOG}</code>。
          </p>
          <FormField
            label="Access 日志路径"
            hint="logging.transports[].path；留空则默认"
            value={accessDisplay}
            onChange={(e) => patchFile(e.target.value, str(levels.error))}
          />
          <FormField
            label="Error 日志路径"
            hint="logging.transports[].levels.error；留空则默认"
            value={errorDisplay}
            onChange={(e) => patchFile(str(file.path), e.target.value)}
          />
        </FormSection>
      )}
    </FormGrid>
  )
}

function MaintenanceModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const form = globalMaintenanceFromDoc(doc)
  const patch = (next: GlobalMaintenanceForm) => onChange(patchGlobalMaintenance(doc, next))
  return <GlobalMaintenanceFormFields form={form} onChange={patch} />
}

function WAFModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const waf = { ...obj(doc.waf) }
  const wafEnabled = bool(waf.enabled, false)
  const patchWaf = (fn: (next: Record<string, unknown>) => void) => {
    const next = { ...waf }
    fn(next)
    onChange({ waf: next })
  }

  return (
    <FormGrid columns={1}>
      <CollapsibleFormSection
        title="WAF"
        open={wafEnabled}
        onOpenChange={(open) => patchWaf((n) => { n.enabled = open })}
      >
        <FormSelectField
          label="全局处置"
          keyName="waf.log_only"
          hint="未单独设置 action 的规则默认跟随此项；写入 log_only: true 表示记录"
          value={wafGlobalModeFromLogOnly(bool(waf.log_only))}
          onChange={(e) => {
            const mode = e.target.value as WAFGlobalMode
            patchWaf((n) => {
              if (mode === 'audit') n.log_only = true
              else delete n.log_only
            })
          }}
        >
          {WAF_GLOBAL_MODE_OPTIONS.map((o) => (
            <option key={o.value} value={o.value} title={o.hint}>
              {o.label} — {o.hint}
            </option>
          ))}
        </FormSelectField>
        <p className="form-hint">
          单条规则选「继承」时跟随全局处置；可显式覆盖为 拦截 / 仅记录 / 通过。
        </p>
        <FormCheckbox
          label="禁用内置规则"
          checked={bool(waf.disable_builtin)}
          onChange={(v) => patchWaf((n) => setBool(n, 'disable_builtin', v))}
        />
        <FormField
          label="拦截状态码"
          keyName="block_status_code"
          hint="默认 403"
          type="number"
          value={num(waf.block_status_code, 403)}
          onChange={(e) => patchWaf((n) => setNum(n, 'block_status_code', Number(e.target.value)))}
        />
        <WafRulesEditor doc={doc} onChange={onChange} />
      </CollapsibleFormSection>
    </FormGrid>
  )
}

function HealthcheckModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const hc = { ...obj(doc.healthcheck) }
  const outer = { ...obj(hc.outer) }
  const inner = { ...obj(hc.inner) }

  const patch = (outerPatch?: (o: Record<string, unknown>) => void, innerPatch?: (i: Record<string, unknown>) => void) => {
    const nextOuter = { ...outer }
    const nextInner = { ...inner }
    outerPatch?.(nextOuter)
    innerPatch?.(nextInner)
    onChange({ healthcheck: { ...hc, outer: nextOuter, inner: nextInner } })
  }

  return (
    <FormGrid columns={1}>
      <FormSection title="外部探针">
        <FormCheckbox
          label="启用"
          checked={bool(outer.enable, true)}
          onChange={(v) => patch((o) => { o.enable = v })}
        />
        <FormField
          label="路径"
          keyName="path"
          value={str(outer.path, '/healthz')}
          onChange={(e) => patch((o) => setStr(o, 'path', e.target.value))}
        />
        <FormCheckbox
          label="始终返回 OK"
          checked={bool(outer.ok, true)}
          onChange={(v) => patch((o) => { if (v) o.ok = true; else delete o.ok })}
        />
      </FormSection>
      <FormSection title="上游健康检查">
        <FormCheckbox
          label="启用"
          checked={bool(inner.enable, true)}
          onChange={(v) => patch(undefined, (i) => { i.enable = v })}
        />
        <FormField
          label="间隔（秒）"
          keyName="interval"
          type="number"
          value={num(inner.interval, 30)}
          onChange={(e) => patch(undefined, (i) => setNum(i, 'interval', Number(e.target.value)))}
        />
        <FormField
          label="超时（秒）"
          keyName="timeout"
          type="number"
          value={num(inner.timeout, 5)}
          onChange={(e) => patch(undefined, (i) => setNum(i, 'timeout', Number(e.target.value)))}
        />
      </FormSection>
    </FormGrid>
  )
}

function HTTPSModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const https = { ...obj(doc.https) }
  const redirect = { ...obj(https.redirect_from_http) }

  const patchHttps = (fn: (next: Record<string, unknown>) => void) => {
    const next = { ...https }
    fn(next)
    onChange({ https: next })
  }

  return (
    <FormGrid columns={1}>
      <FormField
        label="HTTPS 端口"
        keyName="port"
        type="number"
        value={num(https.port, 443)}
        onChange={(e) => patchHttps((n) => setNum(n, 'port', Number(e.target.value)))}
      />
      <FormSection title="HTTP → HTTPS 重定向">
        <FormCheckbox
          label="启用强制重定向"
          checked={bool(redirect.enabled)}
          onChange={(v) =>
            patchHttps((n) => {
              const r = { ...obj(n.redirect_from_http) }
              setBool(r, 'enabled', v)
              n.redirect_from_http = r
            })
          }
        />
        {bool(redirect.enabled) && (
          <FormCheckbox
            label="301/308 永久重定向（默认 302/307）"
            checked={bool(redirect.permanent)}
            onChange={(v) =>
              patchHttps((n) => {
                const r = { ...obj(n.redirect_from_http) }
                setBool(r, 'permanent', v)
                n.redirect_from_http = r
              })
            }
          />
        )}
      </FormSection>
      <FormSection title="HTTP/3">
        <FormCheckbox
          label="启用 HTTP/3"
          checked={bool(https.enable_http3)}
          onChange={(v) => patchHttps((n) => setBool(n, 'enable_http3', v))}
        />
        <FormField
          label="HTTP/3 UDP 端口"
          keyName="http3_port"
          hint="0 表示与 HTTPS 同端口"
          type="number"
          value={num(https.http3_port, 0)}
          onChange={(e) => patchHttps((n) => setNum(n, 'http3_port', Number(e.target.value)))}
        />
      </FormSection>
      <SslCertsEditor doc={doc} onChange={onChange} />
    </FormGrid>
  )
}

function RulesModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  return (
    <FormGrid columns={1}>
      <RulesEditor doc={doc} onChange={onChange} />
    </FormGrid>
  )
}

function ServicesModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  return (
    <FormGrid columns={1}>
      <ServicesEditor doc={doc} onChange={onChange} />
    </FormGrid>
  )
}

function FallbackModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  return <FallbackEditor doc={doc} onChange={onChange} />
}

function YamlFallbackForm({
  yamlText,
  onChange,
  hint,
}: {
  yamlText: string
  onChange: (yaml: string) => void
  hint: string
}) {
  return (
    <FormGrid columns={1}>
      <p className="form-hint">{hint}</p>
      <textarea
        className="code config-module-text form-control"
        spellCheck={false}
        value={yamlText}
        onChange={(e) => onChange(e.target.value)}
      />
    </FormGrid>
  )
}

export function ConfigModuleForm({
  moduleId,
  moduleYAML,
  onYAMLChange,
  onSwitchToYaml,
}: {
  moduleId: string
  moduleYAML: string
  onYAMLChange: (yaml: string) => void
  onSwitchToYaml?: () => void
}) {
  const doc = parseModuleDoc(moduleYAML)
  const onDocChange = (next: Record<string, unknown>) => onYAMLChange(stringifyModuleDoc(next))

  switch (moduleId) {
    case 'general':
      return <GeneralModuleForm doc={doc} onChange={onDocChange} />
    case 'cache':
      return <CacheModuleForm doc={doc} onChange={onDocChange} />
    case 'admin':
      return <AdminModuleForm doc={doc} onChange={onDocChange} />
    case 'logging':
      return <LoggingModuleForm doc={doc} onChange={onDocChange} />
    case 'waf':
      return <WAFModuleForm doc={doc} onChange={onDocChange} />
    case 'maintenance':
      return <MaintenanceModuleForm doc={doc} onChange={onDocChange} />
    case 'rate_limit':
      return <RateLimitModuleForm doc={doc} onChange={onDocChange} />
    case 'security':
      return <SecurityModuleForm doc={doc} onChange={onDocChange} />
    case 'healthcheck':
      return <HealthcheckModuleForm doc={doc} onChange={onDocChange} />
    case 'https':
      return <HTTPSModuleForm doc={doc} onChange={onDocChange} />
    case 'rules':
      return (
        <>
          <RulesModuleForm doc={doc} onChange={onDocChange} />
          {onSwitchToYaml && (
            <button type="button" className="btn btn-ghost config-yaml-link" onClick={onSwitchToYaml}>
              高级：在 YAML 模式中编辑 rules[].waf、rules[].security、rules[].rate_limit、request 等
            </button>
          )}
        </>
      )
    case 'services':
      return <ServicesModuleForm doc={doc} onChange={onDocChange} />
    case 'fallback':
      return <FallbackModuleForm doc={doc} onChange={onDocChange} />
    case 'other':
      return (
        <YamlFallbackForm
          yamlText={moduleYAML}
          onChange={onYAMLChange}
          hint="该模块结构较灵活，暂提供 YAML 编辑。"
        />
      )
    default:
      return (
        <YamlFallbackForm
          yamlText={moduleYAML}
          onChange={onYAMLChange}
          hint="未知模块，使用 YAML 编辑。"
        />
      )
  }
}
