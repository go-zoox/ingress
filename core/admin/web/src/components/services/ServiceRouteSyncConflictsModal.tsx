import { useEffect, useState } from 'react'
import { Drawer } from '../Drawer'
import type { ServiceRouteConflict, ServiceRouteSyncResolution } from '../../lib/serviceRouteSync'

export function ServiceRouteSyncConflictsModal({
  open,
  conflicts,
  onClose,
  onConfirm,
}: {
  open: boolean
  conflicts: ServiceRouteConflict[]
  onClose: () => void
  onConfirm: (resolutions: Record<string, ServiceRouteSyncResolution>) => void
}) {
  const [choices, setChoices] = useState<Record<string, ServiceRouteSyncResolution>>({})

  useEffect(() => {
    if (open) setChoices({})
  }, [open, conflicts])

  if (!open || conflicts.length === 0) return null

  const setAll = (choice: ServiceRouteSyncResolution) => {
    const next: Record<string, ServiceRouteSyncResolution> = {}
    for (const c of conflicts) next[c.id] = choice
    setChoices(next)
  }

  const setOne = (id: string, choice: ServiceRouteSyncResolution) => {
    setChoices((prev) => ({ ...prev, [id]: choice }))
  }

  const allChosen = conflicts.every((c) => choices[c.id] === 'overwrite' || choices[c.id] === 'keep')

  return (
    <Drawer
      open={open}
      title="路由 backend 与目录不一致"
      onClose={onClose}
      width={920}
      footer={
        <>
          <button type="button" className="btn" onClick={onClose}>
            取消
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!allChosen}
            onClick={() => onConfirm(choices)}
          >
            继续
          </button>
        </>
      }
    >
      <p style={{ marginTop: 0, color: 'var(--text-muted)', fontSize: 13 }}>
        以下路由引用了正在变更的服务，但 backend 中的 port / protocol / mode / healthcheck
        与<strong>变更前</strong>的服务目录不一致（可能在配置中心手动改过）。请选择是保留路由上的改动，还是用新目录覆盖。
      </p>
      <div className="toolbar" style={{ marginBottom: 12 }}>
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => setAll('keep')}>
          全部保留路由改动
        </button>
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => setAll('overwrite')}>
          全部覆盖为目录
        </button>
      </div>
      <div className="panel-table-wrap">
        <table className="data compact">
          <thead>
            <tr>
              <th>路由</th>
              <th>服务</th>
              <th>当前 backend</th>
              <th>处理方式</th>
            </tr>
          </thead>
          <tbody>
            {conflicts.map((c) => (
              <tr key={c.id}>
                <td>{c.label}</td>
                <td>
                  <code>{c.serviceName}</code>
                </td>
                <td>
                  <code>{c.routeSummary}</code>
                </td>
                <td>
                  <label className="inline-radio">
                    <input
                      type="radio"
                      name={`sync-${c.id}`}
                      checked={choices[c.id] === 'keep'}
                      onChange={() => setOne(c.id, 'keep')}
                    />
                    保留
                  </label>
                  <label className="inline-radio" style={{ marginLeft: 12 }}>
                    <input
                      type="radio"
                      name={`sync-${c.id}`}
                      checked={choices[c.id] === 'overwrite'}
                      onChange={() => setOne(c.id, 'overwrite')}
                    />
                    覆盖
                  </label>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Drawer>
  )
}
