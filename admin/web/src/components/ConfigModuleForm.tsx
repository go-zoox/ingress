import {
  FormCheckbox,
  FormField,
  FormGrid,
  FormSection,
  FormSelectField,
} from './Form'
import { FallbackEditor } from './config/FallbackEditor'
import { RulesEditor } from './config/RulesEditor'
import { SslCertsEditor } from './config/SslCertsEditor'
import { WafRulesEditor } from './config/WafRulesEditor'
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
        label="配置版本"
        keyName="version"
        value={str(doc.version, 'v1')}
        onChange={(e) => patch((n) => setStr(n, 'version', e.target.value))}
      />
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
          label="404 页面暴露请求细节"
          checked={bool(doc.error_page_expose_details)}
          onChange={(v) => patch((n) => setBool(n, 'error_page_expose_details', v))}
        />
      </FormSection>
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

function LoggingModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const logging = { ...obj(doc.logging) }
  const transports = arr<Record<string, unknown>>(logging.transports)
  const fileIdx = transports.findIndex((t) => str(t.type) === 'file')
  const file = fileIdx >= 0 ? { ...obj(transports[fileIdx]) } : { type: 'file', path: '', levels: {} }
  const levels = { ...obj(file.levels) }

  const patchLogging = (fn: (next: Record<string, unknown>) => void) => {
    const nextLogging = { ...logging }
    fn(nextLogging)
    onChange({ logging: nextLogging })
  }

  const patchFile = (accessPath: string, errorPath: string) => {
    const nextFile = {
      ...file,
      type: 'file',
      path: accessPath,
      levels: errorPath ? { ...levels, error: errorPath } : { ...levels },
    }
    if (!errorPath) delete (nextFile.levels as Record<string, unknown>).error
    const nextTransports = [...transports]
    if (fileIdx >= 0) nextTransports[fileIdx] = nextFile
    else nextTransports.push(nextFile)
    patchLogging((n) => {
      n.transports = nextTransports
    })
  }

  return (
    <FormGrid>
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
      <FormSection title="文件输出">
        <FormField
          label="Access 日志路径"
          hint="logging.transports[].path"
          value={str(file.path)}
          onChange={(e) => patchFile(e.target.value, str(levels.error))}
        />
        <FormField
          label="Error 日志路径"
          hint="logging.transports[].levels.error"
          value={str(levels.error)}
          onChange={(e) => patchFile(str(file.path), e.target.value)}
        />
      </FormSection>
    </FormGrid>
  )
}

function WAFModuleForm({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const waf = { ...obj(doc.waf) }
  const patchWaf = (fn: (next: Record<string, unknown>) => void) => {
    const next = { ...waf }
    fn(next)
    onChange({ waf: next })
  }

  return (
    <FormGrid columns={1}>
      <FormCheckbox
        label="启用 WAF"
        checked={bool(waf.enabled, true)}
        onChange={(v) => patchWaf((n) => { n.enabled = v })}
      />
      <FormCheckbox
        label="仅审计不拦截"
        checked={bool(waf.log_only)}
        onChange={(v) => patchWaf((n) => { if (v) n.log_only = true; else delete n.log_only })}
      />
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
          label="301/308 永久重定向"
          checked={bool(redirect.permanent, true)}
          onChange={(v) =>
            patchHttps((n) => {
              const r = { ...obj(n.redirect_from_http) }
              if (v) r.permanent = true
              else delete r.permanent
              n.redirect_from_http = r
            })
          }
        />
        <FormCheckbox
          label="禁用强制重定向"
          checked={bool(redirect.disabled)}
          onChange={(v) =>
            patchHttps((n) => {
              const r = { ...obj(n.redirect_from_http) }
              setBool(r, 'disabled', v)
              n.redirect_from_http = r
            })
          }
        />
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
    case 'logging':
      return <LoggingModuleForm doc={doc} onChange={onDocChange} />
    case 'waf':
      return <WAFModuleForm doc={doc} onChange={onDocChange} />
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
              高级：在 YAML 模式中编辑 rules[].waf、auth、request 等
            </button>
          )}
        </>
      )
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
