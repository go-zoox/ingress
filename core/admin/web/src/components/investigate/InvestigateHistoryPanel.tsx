import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Clock, Trash2 } from 'lucide-react'
import { investigateLink } from '../../lib/deepLinks'
import { clearInvestigateHistory, loadInvestigateHistory } from '../../lib/investigateHistory'

type Props = {
  version?: number
}

export function InvestigateHistoryPanel({ version = 0 }: Props) {
  const [tick, setTick] = useState(0)
  const items = useMemo(() => loadInvestigateHistory(), [version, tick])

  if (items.length === 0) {
    return (
      <p className="empty-hint" style={{ margin: 0 }}>
        暂无最近调查记录
      </p>
    )
  }

  return (
    <div className="investigate-history">
      <ul className="investigate-history-list">
        {items.map((e) => (
          <li key={`${e.host}-${e.path}-${e.ts}`}>
            <Link
              to={investigateLink({ host: e.host, path: e.path, method: e.method })}
              className="investigate-history-link"
            >
              <code>{e.host}</code>
              <span className="investigate-history-path">{e.path}</span>
              {e.method ? <span className="investigate-history-meta">{e.method}</span> : null}
            </Link>
            <time className="investigate-history-time">
              <Clock size={12} aria-hidden />
              {formatRel(e.ts)}
            </time>
          </li>
        ))}
      </ul>
      <button
        type="button"
        className="btn btn-ghost btn-sm"
        onClick={() => {
          clearInvestigateHistory()
          setTick((t) => t + 1)
        }}
      >
        <Trash2 size={12} aria-hidden /> 清空
      </button>
    </div>
  )
}

function formatRel(ts: number) {
  const d = Date.now() - ts
  if (d < 60_000) return '刚刚'
  if (d < 3600_000) return `${Math.floor(d / 60_000)} 分钟前`
  if (d < 86400_000) return `${Math.floor(d / 3600_000)} 小时前`
  return new Date(ts).toLocaleDateString('zh-CN')
}
