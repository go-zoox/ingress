import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Filter, X } from 'lucide-react'
import {
  hostLooksLikePattern,
  normalizeScopeInput,
  type RouteScope,
} from '../../lib/routeScope'

type ScopeOption = { name: string; count: number }

type Props = {
  ruleIndex: number
  pathIndex: number
  ruleHost: string
  configPath: string
  scope: RouteScope
  hostOptions?: ScopeOption[]
  pathOptions?: ScopeOption[]
}

function buildSelectOptions(
  rows: ScopeOption[] | undefined,
  current: string,
  extra?: string,
): Array<{ value: string; label: string }> {
  const seen = new Set<string>()
  const out: Array<{ value: string; label: string }> = []

  const add = (value: string, label: string) => {
    const v = value.trim()
    if (!v || seen.has(v)) return
    seen.add(v)
    out.push({ value: v, label })
  }

  if (extra) add(extra, extra)
  for (const row of rows ?? []) {
    add(row.name, `${row.name} (${row.count})`)
  }
  if (current.trim() && !seen.has(current.trim())) {
    add(current.trim(), `${current.trim()} (当前)`)
  }

  return out
}

export function RouteScopeBar({
  ruleIndex,
  pathIndex,
  ruleHost,
  configPath,
  scope,
  hostOptions,
  pathOptions,
}: Props) {
  const navigate = useNavigate()
  const [host, setHost] = useState(scope.host)
  const [path, setPath] = useState(scope.path)
  const [pathMatch, setPathMatch] = useState<'prefix' | 'exact'>(scope.pathMatch)

  useEffect(() => {
    setHost(scope.host)
    setPath(scope.path)
    setPathMatch(scope.pathMatch)
  }, [scope.host, scope.path, scope.pathMatch])

  const hostSelectOptions = useMemo(() => {
    const extra =
      !hostLooksLikePattern(ruleHost) && ruleHost.trim() ? ruleHost : undefined
    return buildSelectOptions(hostOptions, host, extra)
  }, [hostOptions, host, ruleHost])

  const pathSelectOptions = useMemo(() => {
    const extra = configPath.trim() || undefined
    return buildSelectOptions(pathOptions, path, extra)
  }, [pathOptions, path, configPath])

  const hasScope = Boolean(scope.host || scope.path)
  const isPatternHost = hostLooksLikePattern(ruleHost)

  const applyScope = (next: RouteScope) => {
    const q = new URLSearchParams()
    if (next.host) q.set('host', next.host)
    if (next.path) q.set('path', next.path)
    if (next.pathMatch === 'exact') q.set('path_match', 'exact')
    const qs = q.toString()
    navigate(`/routes/${ruleIndex}/${pathIndex}${qs ? `?${qs}` : ''}`, { replace: false })
  }

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const normalized = normalizeScopeInput(host, path)
    applyScope({
      host: normalized.host,
      path: normalized.path,
      pathMatch,
    })
  }

  const onClear = () => {
    setHost('')
    setPath('')
    setPathMatch('prefix')
    navigate(`/routes/${ruleIndex}/${pathIndex}`)
  }

  return (
    <div className="panel route-scope-panel">
      <div className="panel-head">
        <h2>
          <Filter size={16} aria-hidden style={{ marginRight: 6, verticalAlign: -2 }} />
          流量观测范围
        </h2>
        {hasScope ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={onClear}>
            <X size={14} aria-hidden /> 查看整条规则
          </button>
        ) : (
          <span className="chart-hint">
            {isPatternHost
              ? '规则 Host 为模式，可下钻到具体 Host / Path'
              : '可选：只统计某个 Host 或 URL 的流量'}
          </span>
        )}
      </div>
      <div className="panel-body">
        {hasScope ? (
          <p className="route-scope-active">
            当前观测：
            <code>{scope.host || '（全部 Host）'}</code>
            {scope.path ? (
              <>
                {' '}
                <code>{scope.path}</code>
                <span className="route-scope-match-tag">
                  {scope.pathMatch === 'exact' ? '精确' : '前缀'}
                </span>
              </>
            ) : null}
            <span className="route-scope-rule-hint">
              配置 Host：<code>{ruleHost}</code>
            </span>
          </p>
        ) : null}

        <form className="route-scope-form" onSubmit={onSubmit}>
          <label className="route-scope-field">
            <span className="field-label">Host</span>
            <select
              className="form-control route-scope-select"
              value={host}
              onChange={(e) => setHost(e.target.value)}
              aria-label="Host"
            >
              <option value="">不限 Host</option>
              {hostSelectOptions.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
          <label className="route-scope-field route-scope-field-grow">
            <span className="field-label">Path</span>
            <select
              className="form-control route-scope-select"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              aria-label="Path"
            >
              <option value="">不限 Path</option>
              {pathSelectOptions.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
          <label className="route-scope-field route-scope-field-narrow">
            <span className="field-label">Path 匹配</span>
            <select
              className="form-control"
              value={pathMatch}
              onChange={(e) => setPathMatch(e.target.value as 'prefix' | 'exact')}
              aria-label="Path 匹配方式"
            >
              <option value="prefix">前缀</option>
              <option value="exact">精确</option>
            </select>
          </label>
          <button type="submit" className="btn btn-primary btn-sm">
            应用
          </button>
        </form>
      </div>
    </div>
  )
}
