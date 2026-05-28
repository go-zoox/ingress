import { Link } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { OverviewAttentionPanel } from '../components/OverviewAttentionPanel'
import { PageHeader } from '../components/PageHeader'
import { useAttentionData } from '../hooks/useAttentionData'

export function AttentionPage() {
  const { metrics, certs, healthChecks, wafBlocks, parseIssues, loading, refresh, handleParseIssueStatus } =
    useAttentionData({
      parseIssueLimit: 50,
      wafLimit: 30,
    })

  return (
    <div className="page">
      <PageHeader
        title="需要关注"
        desc="待处理项：健康检查 DOWN、WAF 拦截、日志解析异常、证书与质量信号。与「事件」不同，这里强调可操作的处置队列。"
        actions={
          <>
            <Link to="/events" className="btn btn-ghost btn-sm">
              事件流
            </Link>
            <button type="button" className="btn btn-sm" onClick={() => refresh()}>
              刷新
            </button>
          </>
        }
      />

      {loading && !metrics ? (
        <p className="empty-hint">
          <AlertTriangle size={16} style={{ verticalAlign: 'middle', marginRight: 6 }} />
          加载中…
        </p>
      ) : (
        <OverviewAttentionPanel
          metrics={metrics}
          certs={certs}
          healthChecks={healthChecks}
          wafBlocks={wafBlocks}
          parseIssues={parseIssues}
          onParseIssueStatus={handleParseIssueStatus}
          embedded={false}
        />
      )}
    </div>
  )
}
