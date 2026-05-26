import type { MatchPreview } from '../../api/client'

type Props = {
  urlInput: string
  onUrlInputChange: (v: string) => void
  onMatch: () => void
  match: MatchPreview | null
  matchError: string
  onOpenRoute?: (ruleIndex: number, pathIndex: number) => void
}

export function RouteMatchTab({
  urlInput,
  onUrlInputChange,
  onMatch,
  match,
  matchError,
  onOpenRoute,
}: Props) {
  return (
    <div className="panel">
      <div className="panel-head">
        <h2>试匹配</h2>
      </div>
      <div className="panel-body">
        <p className="match-hint">输入完整 URL，自动提取 Host 与 Path 进行匹配。</p>
        {matchError ? <p className="err">{matchError}</p> : null}
        <label className="field-label">URL</label>
        <input
          type="text"
          className="field-input-last"
          placeholder="https://api.example.com/v2/users"
          value={urlInput}
          onChange={(e) => onUrlInputChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') onMatch()
          }}
        />
        <button type="button" className="btn btn-primary" style={{ width: '100%', marginTop: 12 }} onClick={onMatch}>
          试匹配
        </button>

        {match && (
          <div className={`match-result ${match.matched ? 'hit' : 'miss'}`} style={{ marginTop: 16 }}>
            {match.matched ? (
              <>
                <h3>
                  命中规则 #{match.rule_index}
                  {match.fallback && '（fallback）'}
                </h3>
                <dl>
                  <dt>Host</dt>
                  <dd>
                    {match.host}（{match.host_type}）
                  </dd>
                  <dt>Path</dt>
                  <dd>{match.path}</dd>
                  <dt>Backend</dt>
                  <dd>{match.backend_type}</dd>
                  <dt>目标</dt>
                  <dd>
                    <code>{match.target}</code>
                  </dd>
                </dl>
                {onOpenRoute && match.path_index != null ? (
                  <button
                    type="button"
                    className="btn btn-sm"
                    style={{ marginTop: 12 }}
                    onClick={() => onOpenRoute(match.rule_index, match.path_index!)}
                  >
                    打开路由详情
                  </button>
                ) : null}
              </>
            ) : (
              <>
                <h3>未命中</h3>
                <p>{match.message || '将走 fallback 或返回 404'}</p>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
