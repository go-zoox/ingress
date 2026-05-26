import { Link } from 'react-router-dom'
import type { MatchPreview } from '../../api/client'
import { routeDetailLink } from '../../lib/deepLinks'

type Props = {
  match: MatchPreview | null
}

export function InvestigateMatchPanel({ match }: Props) {
  if (!match) {
    return <p className="empty-hint">未执行路由匹配</p>
  }

  return (
    <div className={`match-result ${match.matched ? 'hit' : 'miss'}`}>
      {match.matched ? (
        <>
          <h3>
            命中规则 #{match.rule_index}
            {match.fallback ? '（fallback）' : ''}
          </h3>
          <dl>
            <dt>Host</dt>
            <dd>
              {match.host}
              {match.host_type ? `（${match.host_type}）` : ''}
            </dd>
            <dt>Path</dt>
            <dd>
              <code>{match.path}</code>
            </dd>
            <dt>Backend</dt>
            <dd>{match.backend_type}</dd>
            <dt>目标</dt>
            <dd>
              <code>{match.target}</code>
            </dd>
          </dl>
          {match.path_index != null && match.path_index >= 0 ? (
            <Link
              to={routeDetailLink(match.rule_index, match.path_index, { host: match.host, path: match.path })}
              className="btn btn-sm"
              style={{ marginTop: 12 }}
            >
              路由详情
            </Link>
          ) : null}
        </>
      ) : (
        <>
          <h3>未命中</h3>
          <p>{match.message || '将走 fallback 或返回 404'}</p>
        </>
      )}
    </div>
  )
}
