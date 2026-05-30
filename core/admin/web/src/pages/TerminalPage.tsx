import { PageHeader } from '../components/PageHeader'
import { WebTerminal } from '../components/terminal/WebTerminal'

export function TerminalPage() {
  return (
    <>
      <PageHeader
        title="Web 终端"
        desc="通过 Xterm 连接 Admin 进程所在主机的交互式 Shell；断线 60 秒内自动重连并恢复同一会话。"
      />
      <section className="panel web-terminal-panel">
        <div className="panel-body">
          <WebTerminal />
        </div>
      </section>
    </>
  )
}
