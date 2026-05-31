import { useEffect, useState } from 'react'
import { PageHeader } from '../components/PageHeader'
import { api, type CacheOverview } from '../api/client'

function HitRankPanel<T extends { hits: number; total: number; hit_rate: number }>({
  title,
  empty,
  rows,
  label,
  labelClassName,
  rowClassName,
}: {
  title: string
  empty: string
  rows: T[]
  label: (row: T) => string
  labelClassName: string
  rowClassName: string
}) {
  return (
    <div className="panel chart-panel">
      <div className="panel-head">
        <h2>{title}</h2>
        <span className="chart-hint">来自 access.log cache_hit=1</span>
      </div>
      <div className="panel-body">
        {rows.length === 0 ? (
          <p className="empty-hint">{empty}</p>
        ) : (
          <div className="ranked-bar-list">
            {rows.map((row) => {
              const text = label(row)
              return (
                <div key={text} className={`bar-row ${rowClassName}`}>
                  <span className={labelClassName} title={text}>
                    {text}
                  </span>
                  <div className="bar-track">
                    <div
                      className="bar-fill seg-2xx"
                      style={{ width: `${Math.min(100, row.hit_rate)}%` }}
                    />
                  </div>
                  <span className="bar-val">
                    {row.hit_rate.toFixed(0)}% ({row.hits}/{row.total})
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

export function CachePage() {
  const [data, setData] = useState<CacheOverview | null>(null)
  const [err, setErr] = useState('')

  useEffect(() => {
    api
      .cacheOverview()
      .then(setData)
      .catch((e: Error) => setErr(e.message))
  }, [])

  const g = data?.global
  const stats = data?.stats
  const routes = data?.routes ?? []

  return (
    <div className="page">
      <PageHeader
        title="缓存"
        desc="全局 cache 后端、路由级 HTTP 响应缓存策略与 access log 命中统计"
      />
      {err && <p className="err">{err}</p>}
      <div className="cards">
        <div className="card">
          <div className="label">引擎</div>
          <div className="value">{g?.engine ?? '—'}</div>
          <div className="sub">{g?.enabled ? '已配置' : '未配置 Redis'}</div>
        </div>
        <div className="card">
          <div className="label">全局 TTL</div>
          <div className="value">{g?.ttl ?? '—'}s</div>
          <div className="sub">cache.ttl</div>
        </div>
        <div className="card">
          <div className="label">Redis</div>
          <div className="value">{g?.host ? `${g.host}:${g.port || 6379}` : 'memory'}</div>
          <div className="sub">prefix {g?.prefix || '—'}</div>
        </div>
        <div className="card ok">
          <div className="label">命中率</div>
          <div className="value">{stats ? `${stats.hit_rate.toFixed(1)}%` : '—'}</div>
          <div className="sub">
            {stats ? `${stats.cache_hits} / ${stats.total_requests} 请求` : 'access.log'}
          </div>
        </div>
        <div className="card">
          <div className="label">缓存路由</div>
          <div className="value">{routes.length}</div>
          <div className="sub">backend.cache.enabled</div>
        </div>
      </div>

      <div className="panel">
        <div className="panel-head">
          <h2>路由缓存规则</h2>
        </div>
        <div className="panel-body panel-table-wrap">
          <table className="data">
            <thead>
              <tr>
                <th>Host</th>
                <th>Path</th>
                <th>Backend</th>
                <th>Target</th>
                <th>TTL</th>
                <th>Key</th>
              </tr>
            </thead>
            <tbody>
              {routes.length === 0 ? (
                <tr>
                  <td colSpan={6} className="empty-hint">
                    未启用 backend.cache
                  </td>
                </tr>
              ) : (
                routes.map((r) => (
                  <tr key={r.id}>
                    <td>{r.host}</td>
                    <td>
                      <code>{r.path}</code>
                    </td>
                    <td>{r.backend_type}</td>
                    <td>{r.target}</td>
                    <td>{r.ttl}s</td>
                    <td>{r.key_hash}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="charts-grid">
        <HitRankPanel
          title="Host 命中排行"
          empty="暂无 Host 统计"
          rows={stats?.top_hosts ?? []}
          label={(h) => h.host}
          labelClassName="bar-label host-label"
          rowClassName="host-rank"
        />
        <HitRankPanel
          title="Path 命中排行"
          empty="暂无 Path 统计"
          rows={stats?.top_paths ?? []}
          label={(p) => p.path}
          labelClassName="bar-label path-label"
          rowClassName="path-rank"
        />
      </div>
    </div>
  )
}
