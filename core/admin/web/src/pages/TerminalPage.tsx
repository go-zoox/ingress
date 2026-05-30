import { PageHeader } from '../components/PageHeader'
import { WebTerminal } from '../components/terminal/WebTerminal'

export function TerminalPage() {
  return (
    <>
      <PageHeader
        title="Web 终端"
        desc="支持多标签并行 Shell；每个标签独立会话，断线 60 秒内可自动重连恢复。"
      />
      <section className="panel web-terminal-panel">
        <div className="panel-body">
          <WebTerminal />
        </div>
      </section>
    </>
  )
}
